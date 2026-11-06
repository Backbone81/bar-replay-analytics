package playback

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"syscall"
	"time"

	"github.com/backbone81/bar-replay-analytics-go/internal/events"
	"github.com/backbone81/bar-replay-analytics-go/internal/newfiles"
	"github.com/backbone81/bar-replay-analytics-go/internal/playback/widget"
)

// analyticsFailed is the analytics widget's message when it cannot open its events file.
const analyticsFailed = "Replay Analytics: cannot write"

// monitorInterval is how often Play looks at the events file and the engine log.
const monitorInterval = time.Second

// lastFrameWindow is how much of the events file's tail monitor.lastFrame reads: several events.
const lastFrameWindow = 64 * 1024

// Result is the outcome of a playback.
type Result struct {
	// EventsFile is the path of the events file the analytics widget wrote.
	EventsFile string

	// LogFile is the path of the engine log.
	LogFile string

	// Duration is the wall clock time the engine ran.
	Duration time.Duration

	// StopReason is why Play stopped the engine, or empty if the engine exited by itself.
	StopReason string

	// ExitErr is the error of the engine's exit (an *exec.ExitError for a non-zero exit code), or nil.
	ExitErr error

	// Events is the number of events in the events file. Is zero if End is nil.
	Events int

	// End is when and why the widget stopped logging: events.EndReasonGameOver if the whole game was played back.
	// Is nil if the events file is incomplete: the engine crashed or was stopped.
	End *events.End
}

// GameOver reports whether the whole game was played back.
func (r *Result) GameOver() bool {
	return r.End != nil && r.End.Reason == events.EndReasonGameOver
}

// Play plays replay back. The analytics widget writes the events file into the playback's write dir, and Play copies
// it next to the replay file at the end (Replay.EventsFilePath), also an incomplete one; the engine log goes there
// directly (Replay.LogFilePath). It returns an error if the engine could not be run, or the analytics widget wrote no
// events file, and otherwise a Result, also if the playback did not reach the end of the game (see Result.GameOver).
// Several Play calls may run at the same time, but not for the same replay file, as their events files would be the
// same. Cancelling ctx stops the engine.
func (p *Player) Play(ctx context.Context, replay *Replay) (*Result, error) {
	result := &Result{
		EventsFile: replay.EventsFilePath(),
		LogFile:    replay.LogFilePath(),
	}
	// A failed playback must not leave the events file of an earlier one behind.
	if err := os.Remove(result.EventsFile); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}

	writeDir, err := p.MakeWriteDir()
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(writeDir) //nolint:errcheck

	logFile, err := os.OpenFile(result.LogFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, filePermissions)
	if err != nil {
		return nil, err
	}

	writtenEventsFile := p.WriteDirEventsFile(writeDir, replay)
	start := time.Now()
	err = p.runEngine(ctx, replay, writeDir, writtenEventsFile, logFile, result)
	result.Duration = time.Since(start)
	copyErr := copyFileAtomically(writtenEventsFile, result.EventsFile)
	if errors.Is(copyErr, fs.ErrNotExist) {
		copyErr = nil // reported below, after the engine's own errors
	}
	if err := errors.Join(err, logFile.Close(), copyErr, ctx.Err()); err != nil {
		return nil, err
	}

	if result.Events, result.End, err = p.readEventsSummary(result.EventsFile); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("no events file was written; see %s", result.LogFile)
		}
		return nil, err
	}
	return result, nil
}

// MakeWriteDir creates a playback's write dir, a temporary folder the caller removes. It holds a copy of the data
// dir's archive cache (so the engine does not rescan every archive), the settings of writeSettings, and the embedded
// analytics widget, which writes its events file into events/ in its folder (see WriteDirEventsFile).
//
// The widget goes into the same folder name as an installed one: the engine lists a path present in both the write
// dir and a data dir once, from the write dir, so the embedded widget replaces the installed one. Under another name
// BAR would load both and reject the second as a duplicate.
func (p *Player) MakeWriteDir() (string, error) {
	writeDir, err := os.MkdirTemp("", "bar-replay-analytics-*")
	if err != nil {
		return "", err
	}

	if err := p.fillWriteDir(writeDir); err != nil {
		_ = os.RemoveAll(writeDir)
		return "", err
	}
	return writeDir, nil
}

// fillWriteDir implements MakeWriteDir for the folder writeDir.
func (p *Player) fillWriteDir(writeDir string) error {
	archiveCaches, err := filepath.Glob(filepath.Join(p.dataDir, "cache", "ArchiveCache*.lua"))
	if err != nil {
		return err
	}

	if len(archiveCaches) != 0 {
		if err := os.Mkdir(filepath.Join(writeDir, "cache"), dirPermissions); err != nil {
			return err
		}
	}

	for _, archiveCache := range archiveCaches {
		if err := copyFile(archiveCache, filepath.Join(writeDir, "cache", filepath.Base(archiveCache))); err != nil {
			return err
		}
	}

	if err := p.writeSettings(writeDir); err != nil {
		return err
	}

	_, err = newfiles.Write(widget.Template, p.writeDirWidgetDir(writeDir))
	return err
}

// writeDirWidgetDir returns the folder of the embedded analytics widget in the write dir writeDir.
func (p *Player) writeDirWidgetDir(writeDir string) string {
	return filepath.Join(writeDir, widgetsDir, filepath.Base(p.analyticsDir))
}

// WriteDirEventsFile returns the path the analytics widget writes the events file of replay to in the write dir
// writeDir: events/<replay name>_events.json in its folder. The engine runs in the write dir, so the widget's relative
// path ends up there.
func (p *Player) WriteDirEventsFile(writeDir string, replay *Replay) string {
	return filepath.Join(p.writeDirWidgetDir(writeDir), "events", replay.EventsFileName())
}

// writeSettings writes springsettings.cfg into writeDir: Config.WorkerThreadCount, and HostPortDefault 0.
//
// A replay is played by a local server that binds a UDP port for spectators. It ignores the start script's HostPort
// and binds HostPortDefault, 8452 unless set (rts/Game/PreGame.cpp StartServerForDemo, rts/Game/ClientSetup.cpp), so
// parallel playbacks would collide ("bind: Address already in use"); 0 lets the system pick a free port.
func (p *Player) writeSettings(writeDir string) error {
	settings := "HostPortDefault = 0\n"
	if p.config.WorkerThreadCount > 0 {
		settings += fmt.Sprintf("WorkerThreadCount = %d\n", p.config.WorkerThreadCount)
	}

	return os.WriteFile(filepath.Join(writeDir, "springsettings.cfg"), []byte(settings), filePermissions)
}

// runEngine runs the engine for replay in writeDir with its output to logFile until it exits, ctx is cancelled or
// monitor.check stops it, and sets Result.StopReason and Result.ExitErr. eventsFile is where the widget writes the
// events, which monitor.check watches. It returns an error only if the engine could not be started.
func (p *Player) runEngine(
	ctx context.Context, replay *Replay, writeDir string, eventsFile string, logFile *os.File, result *Result,
) error {
	engineCtx, stopEngine := context.WithCancel(ctx)
	defer stopEngine()

	//nolint:gosec // The engine Prepare found; the replay path is absolute, so it cannot be taken for an option.
	command := exec.CommandContext(engineCtx, replay.Engine, "--isolation", "--write-dir", writeDir, replay.FilePath)
	command.Dir = writeDir
	command.Env = append(os.Environ(), "SPRING_DATADIR="+p.dataDir)
	command.Stdout = logFile
	command.Stderr = logFile
	command.Cancel = func() error {
		// Ask the engine to quit first; Windows has no SIGTERM, so it is killed after WaitDelay there.
		return command.Process.Signal(syscall.SIGTERM)
	}
	command.WaitDelay = p.config.StopTimeout

	if err := command.Start(); err != nil {
		return err
	}
	exited := make(chan error, 1)
	go func() {
		exited <- command.Wait()
	}()

	monitor := &monitor{
		player:     p,
		replay:     replay,
		eventsFile: eventsFile,
		logFile:    result.LogFile,
		start:      time.Now(),
	}
	monitor.lastChange = monitor.start
	monitor.lastProgress = monitor.start
	ticker := time.NewTicker(monitorInterval)
	defer ticker.Stop()

	for {
		select {
		case exitErr := <-exited:
			if result.StopReason == "" && ctx.Err() == nil {
				// An engine we stopped exits with an error that tells nothing.
				result.ExitErr = exitErr
			}
			return nil
		case now := <-ticker.C:
			if result.StopReason != "" {
				continue
			}
			if result.StopReason = monitor.check(now); result.StopReason != "" {
				stopEngine()
			}
		}
	}
}

// LastFrameIn returns the frame of the last complete event in data, the tail of an events file (one event per line),
// or 0 if there is none. Lines cut off by the tail's start or still being written are skipped.
func LastFrameIn(data []byte) int {
	lines := bytes.Split(data, []byte("\n"))
	for _, line := range slices.Backward(lines) {
		line := bytes.TrimSuffix(bytes.TrimPrefix(bytes.TrimSpace(line), []byte(",")), []byte(","))
		var event struct {
			Frame *int `json:"frame"`
		}
		if json.Unmarshal(line, &event) == nil && event.Frame != nil {
			return *event.Frame
		}
	}
	return 0
}

// readEventsSummary returns the number of events and the end of eventsFile. An incomplete file (invalid JSON, no
// "end") is no error: it returns 0 and nil.
func (p *Player) readEventsSummary(eventsFile string) (int, *events.End, error) {
	data, err := os.ReadFile(eventsFile) //nolint:gosec // The events file of the playback.
	if err != nil {
		return 0, nil, err
	}

	var summary struct {
		Events []json.RawMessage `json:"events"`
		End    *events.End       `json:"end"`
	}
	if json.Unmarshal(data, &summary) != nil || summary.End == nil {
		return 0, nil, nil //nolint:nilerr // An incomplete file is a Result, not an error.
	}
	return len(summary.Events), summary.End, nil
}

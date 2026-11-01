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
)

// analyticsFailed is the analytics widget's message when it cannot open its events file.
const analyticsFailed = "Replay Analytics: cannot write"

// monitorInterval is how often Play looks at the events file and the engine log.
const monitorInterval = time.Second

// lastFrameWindow is how much of the end of the events file lastFrame reads: large enough for several events.
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

// Play plays replay back. It returns an error if the engine could not be run, or the analytics widget wrote no events
// file, and otherwise a Result, also if the playback did not reach the end of the game (see Result.GameOver). Several
// Play calls may run at the same time, but not for replays of the same Name, as their events files would be the same.
// Cancelling ctx stops the engine.
func (p *Player) Play(ctx context.Context, replay *Replay) (*Result, error) {
	eventsDir := p.EventsDir()
	if err := os.MkdirAll(eventsDir, dirPermissions); err != nil {
		return nil, err
	}

	result := &Result{
		EventsFile: filepath.Join(eventsDir, replay.EventsFileName()),
		LogFile:    filepath.Join(eventsDir, replay.LogFileName()),
	}
	if err := os.Remove(result.EventsFile); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}

	writeDir, err := p.makeWriteDir()
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(writeDir) //nolint:errcheck

	logFile, err := os.OpenFile(result.LogFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, filePermissions)
	if err != nil {
		return nil, err
	}

	start := time.Now()
	err = p.runEngine(ctx, replay, writeDir, logFile, result)
	result.Duration = time.Since(start)
	if err := errors.Join(err, logFile.Close(), ctx.Err()); err != nil {
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

// makeWriteDir creates the write dir of a playback: a temporary folder with a copy of the data dir's archive cache, so
// the engine does not scan all archives again, settings of its own (see writeSettings), and a link to the analytics
// widget's events folder, so its events file lands there. The engine reads the data dir as an additional data dir.
// The caller removes the folder.
func (p *Player) makeWriteDir() (string, error) {
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

// fillWriteDir implements makeWriteDir for the folder writeDir.
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

	widgetDir := filepath.Join(writeDir, widgetsDir, filepath.Base(p.analyticsDir))
	if err := os.MkdirAll(widgetDir, dirPermissions); err != nil {
		return err
	}
	return os.Symlink(p.EventsDir(), filepath.Join(widgetDir, "events"))
}

// writeSettings writes the engine settings of a playback into its write dir writeDir: Config.WorkerThreadCount, and
// HostPortDefault 0, so the system picks a free port.
//
// For a replay the engine starts a local server which plays it, with the viewer as a local client that does not
// use the network; the server still binds a UDP port, for other players who might join to watch. The engine drops
// the start script's HostPort and binds HostPortDefault (rts/Game/PreGame.cpp StartServerForDemo,
// rts/Game/ClientSetup.cpp), 8452 unless set, so parallel playbacks would all bind the same port, and all but the
// first exit with "bind: Address already in use".
func (p *Player) writeSettings(writeDir string) error {
	settings := "HostPortDefault = 0\n"
	if p.config.WorkerThreadCount > 0 {
		settings += fmt.Sprintf("WorkerThreadCount = %d\n", p.config.WorkerThreadCount)
	}

	return os.WriteFile(filepath.Join(writeDir, "springsettings.cfg"), []byte(settings), filePermissions)
}

// runEngine runs the engine for replay in writeDir with its output to logFile, until it exits, ctx is cancelled or
// the playback hangs (see monitor), and sets Result.StopReason and Result.ExitErr. It returns an error if the engine
// could not be started.
func (p *Player) runEngine(
	ctx context.Context, replay *Replay, writeDir string, logFile *os.File, result *Result,
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
		eventsFile: result.EventsFile,
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
				// Otherwise the engine was stopped, and its exit error tells nothing.
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

// LastFrameIn returns the frame of the last complete event in data, the end of an events file, or 0 if it holds
// none. The first and last line may be cut off or still being written; such lines are skipped.
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

// readEventsSummary returns the number of events and the end of the events file eventsFile. A file which is not
// valid JSON is incomplete: it has no end and the number of events is not known, so both are zero.
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
		return 0, nil, nil //nolint:nilerr // we want to have a nil error here
	}
	return len(summary.Events), summary.End, nil
}

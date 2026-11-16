package player

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/backbone81/bar-replay-analytics/internal/events"
	"github.com/backbone81/bar-replay-analytics/internal/replay/player/widget"
	"github.com/backbone81/bar-replay-analytics/internal/utils"
)

// Player plays one replay at a time: Prepare makes sure its assets are there, Run plays it back. A Player is not safe
// for concurrent use, as Run keeps the write dir of the running playback; use one Player per goroutine.
type Player struct {
	config Config

	// writeDir is the temporary write dir of the playback Run is running.
	writeDir string
}

// New returns a Player for DefaultConfig changed by options.
func New(options ...Option) (*Player, error) {
	config := DefaultConfig
	for _, option := range options {
		option(&config)
	}
	// The engine runs in the write dir, so a relative data dir would point there.
	var err error
	config.DataDir, err = filepath.Abs(config.DataDir)
	if err != nil {
		return nil, err
	}
	if err := config.Validate(); err != nil {
		return nil, err
	}
	return &Player{
		config: config,
	}, nil
}

// Prepare makes sure the engine, map and game of the replay replayInfo describes (from ReadReplayInfo) are in the data
// dir, downloading what is missing.
func (p *Player) Prepare(ctx context.Context, replayInfo *utils.ReplayInfo) error {
	// The map and the game are downloaded with the pr-downloader of the replay's engine, so the engine comes first.
	if err := p.config.AssetManager.EnsureEngineVersion(ctx, replayInfo.EngineVersion); err != nil {
		return err
	}
	if err := p.config.AssetManager.EnsureMap(ctx, replayInfo.EngineVersion, replayInfo.MapName); err != nil {
		return err
	}
	return p.config.AssetManager.EnsureGame(ctx, replayInfo.EngineVersion, replayInfo.GameType)
}

// Run plays the replay replayInfo describes back and copies the events file the analytics widget wrote next to the
// replay. The engine's output goes to <replay name>.log next to the replay. Canceling ctx stops the engine.
func (p *Player) Run(ctx context.Context, replayInfo *utils.ReplayInfo) error {
	logger := p.config.Logger.With(
		"file-path", p.shortReplayPath(replayInfo.FilePath),
		"date", p.replayDate(replayInfo),
		"map-name", replayInfo.MapName,
		"engine-version", replayInfo.EngineVersion,
		"game-type", replayInfo.GameType,
		"game-time", replayInfo.Duration.Round(time.Second),
	)
	logger.Info("Starting replay")
	start := time.Now()
	defer func() {
		logger.Info("Finished replay", "duration", time.Since(start).Round(time.Second))
	}()

	// The engine writes its settings, caches and logs into its write dir. A temporary one per playback leaves the data
	// dir (the player's install) unchanged and keeps parallel playbacks apart.
	var err error
	p.writeDir, err = os.MkdirTemp("", "bar-replay-analytics-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(p.writeDir) //nolint:errcheck

	if err := p.prepareWriteDir(); err != nil {
		return fmt.Errorf("preparing write dir: %w", err)
	}
	if err := p.runSpringHeadless(ctx, replayInfo.EngineVersion, replayInfo.FilePath); err != nil {
		return fmt.Errorf("running spring-headless: %w", err)
	}

	// The widget can only write under LuaUI/ of the write dir, which is removed, so the events file is copied out. An
	// incomplete one is not: it would look like the analysis of the whole game.
	eventsFilePath := utils.DemoEventsFilePath(replayInfo.FilePath)
	eventsFileName := filepath.Base(eventsFilePath)
	writtenEventsFilePath := filepath.Join(utils.AnalyticsWidgetEventsDir(p.writeDir), eventsFileName)
	if err := events.ValidateEventsFile(writtenEventsFilePath); err != nil {
		return fmt.Errorf("checking the events file: %w", err)
	}
	return utils.CopyFile(writtenEventsFilePath, eventsFilePath)
}

// prepareWriteDir fills the new write dir with what the engine needs from it for a playback.
func (p *Player) prepareWriteDir() error {
	if err := p.prepareWriteDirArchiveCache(); err != nil {
		return fmt.Errorf("preparing archive cache: %w", err)
	}
	if err := p.prepareWriteDirSpringSettings(); err != nil {
		return fmt.Errorf("preparing spring settings: %w", err)
	}
	if err := p.prepareWriteDirWidget(); err != nil {
		return fmt.Errorf("preparing widget: %w", err)
	}
	if err := p.prepareWriteDirWidgetConfig(); err != nil {
		return fmt.Errorf("preparing widget config: %w", err)
	}
	return nil
}

// prepareWriteDirArchiveCache copies the data dir's archive cache into the write dir. The engine keeps it in the write
// dir only; without it, it would scan and checksum every map and game archive of the data dir before each playback.
func (p *Player) prepareWriteDirArchiveCache() error {
	archiveCaches, err := filepath.Glob(filepath.Join(utils.CacheDir(p.config.DataDir), "ArchiveCache*.lua"))
	if err != nil {
		return err
	}

	//nolint:gosec // This directory does not contain sensitive data
	if err := os.MkdirAll(
		utils.CacheDir(p.writeDir),
		0o755,
	); err != nil {
		return err
	}
	for _, archiveCache := range archiveCaches {
		if err := utils.CopyFile(
			archiveCache,
			filepath.Join(utils.CacheDir(p.writeDir), filepath.Base(archiveCache)),
		); err != nil {
			return err
		}
	}
	return nil
}

// prepareWriteDirSpringSettings writes the springsettings.cfg that runSpringHeadless passes as the engine's only config
// file (--config). Otherwise the engine reads the springsettings.cfg of every data dir and rewrites them as it starts
// (ConfigHandlerImpl::FinalizeLoad): the player's own settings would apply to the playback and be changed by it.
func (p *Player) prepareWriteDirSpringSettings() error {
	lines := []string{
		// The engine hosts a local server for the replay. Port 0 lets the OS pick a free one, so parallel playbacks do
		// not collide.
		"HostPortDefault = 0",
	}
	if p.config.WorkerThreadCount > 0 {
		lines = append(lines, fmt.Sprintf("WorkerThreadCount = %d", p.config.WorkerThreadCount))
	}
	return utils.WriteFile(utils.SpringsettingsCfgFilePath(p.writeDir), []byte(strings.Join(lines, "\n")), 0o644)
}

// prepareWriteDirWidget writes the embedded analytics widget into the write dir's user widgets folder, so the playback
// loads it without it being installed in the data dir.
func (p *Player) prepareWriteDirWidget() error {
	if err := utils.WriteTemplate(
		slog.New(slog.DiscardHandler),
		widget.Template,
		utils.AnalyticsWidgetDir(p.writeDir),
	); err != nil {
		return err
	}
	return nil
}

// prepareWriteDirWidgetConfig writes the widget config file into the write dir with the analytics widget and the user
// widgets of the data dir in its order list (see WidgetConfig), so games whose widget handler enables only the user
// widgets in that list run them too.
func (p *Player) prepareWriteDirWidgetConfig() error {
	nameLiterals, err := UserWidgetNameLiterals(utils.WidgetsDir(p.config.DataDir))
	if err != nil {
		return fmt.Errorf("reading the names of the user widgets: %w", err)
	}

	filePath := utils.WidgetHandlerConfigFilePath(p.writeDir)
	//nolint:gosec // This directory does not contain sensitive data
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		return err
	}
	return utils.WriteFile(filePath, WidgetConfig(append(nameLiterals, `"`+widget.Name+`"`)), 0o644)
}

// runSpringHeadless plays the replay filePath with the engine engineVersion. The engine's output replaces <replay
// name>.log next to the replay, which the error names if the engine fails. A Watchdog stops the engine when the
// analytics widget gives no sign of life; there is no fixed time limit, as big games take long at full speed too.
func (p *Player) runSpringHeadless(ctx context.Context, engineVersion string, filePath string) error {
	//nolint:gosec // This file does not contain sensitive data
	logFile, err := os.OpenFile(
		utils.DemoLogFilePath(filePath),
		os.O_WRONLY|os.O_CREATE|os.O_TRUNC,
		0o644,
	)
	if err != nil {
		return err
	}
	defer logFile.Close() //nolint:errcheck

	// The defers run in reverse: cancel ends Watch, then Wait waits for it, so no goroutine outlives the playback.
	var watchdogWaitGroup sync.WaitGroup
	defer watchdogWaitGroup.Wait()

	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)

	watchdog := NewWatchdog(p.config.WidgetStartTimeout, p.config.WidgetHeartbeatTimeout)
	watchdogWaitGroup.Go(func() {
		watchdog.Watch(ctx, cancel)
	})

	//nolint:gosec // The user does not control the command input.
	command := exec.CommandContext(ctx, utils.SpringHeadlessFilePath(p.config.DataDir, engineVersion),
		// Isolation keeps the engine from adding the home and system data dirs (DataDirLocater::LocateDataDirs): it
		// uses only the write dir, the engine dir and SPRING_DATADIR.
		"--isolation",
		"--write-dir", p.writeDir,
		"--config", utils.SpringsettingsCfgFilePath(p.writeDir),
		filePath,
	)
	// The widget writes its events file relative to the working directory.
	command.Dir = p.writeDir
	// The data dir is added read only: the engine writes into its write dir only.
	command.Env = append(os.Environ(), "SPRING_DATADIR="+p.config.DataDir)
	command.Cancel = func() error {
		return terminate(command.Process)
	}
	// The engine is killed if it has not quit WaitDelay after terminate.
	command.WaitDelay = 5 * time.Second
	output := io.MultiWriter(logFile, watchdog)
	command.Stdout = output
	command.Stderr = output

	if err := command.Run(); err != nil {
		return errors.Join(
			fmt.Errorf("running the replay (see %s): %w", utils.DemoLogFilePath(filePath), err),
			context.Cause(ctx),
		)
	}
	return nil
}

func (p *Player) shortReplayPath(filePath string) string {
	return utils.RelativePath(utils.DemosDir(p.config.DataDir), filePath)
}

func (p *Player) replayDate(replayInfo *utils.ReplayInfo) string {
	return replayInfo.StartTime.Format("2006.01.02")
}

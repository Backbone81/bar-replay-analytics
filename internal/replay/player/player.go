package player

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/backbone81/bar-replay-analytics-go/internal/events"
	"github.com/backbone81/bar-replay-analytics-go/internal/replay/player/widget"
	"github.com/backbone81/bar-replay-analytics-go/internal/utils"
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
// dir, downloading what is missing, and returns a copy of replayInfo with EngineDir set.
func (p *Player) Prepare(ctx context.Context, replayInfo *ReplayInfo) (*ReplayInfo, error) {
	// The map and the game are downloaded with the pr-downloader of the replay's engine, so the engine comes first.
	engineDir, err := p.config.AssetManager.EnsureEngineVersion(ctx, replayInfo.EngineVersion)
	if err != nil {
		return nil, err
	}
	if err := p.config.AssetManager.EnsureMap(ctx, replayInfo.EngineVersion, replayInfo.MapName); err != nil {
		return nil, err
	}
	if err := p.config.AssetManager.EnsureGame(ctx, replayInfo.EngineVersion, replayInfo.GameType); err != nil {
		return nil, err
	}

	result := *replayInfo
	result.EngineDir = engineDir
	return &result, nil
}

// Run plays the replay replayInfo describes back and copies the events file the analytics widget wrote next to the
// replay. The engine's output goes to <replay name>.log next to the replay. Canceling ctx stops the engine.
func (p *Player) Run(ctx context.Context, replayInfo *ReplayInfo) error {
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
	if err := p.runSpringHeadless(ctx, replayInfo.EngineDir, replayInfo.FilePath); err != nil {
		return fmt.Errorf("running spring-headless: %w", err)
	}

	// The widget can only write under LuaUI/ of the write dir, which is removed, so the events file is copied out. An
	// incomplete one is not: it would look like the analysis of the whole game.
	eventsFilePath := EventsFilePath(replayInfo.FilePath)
	eventsFileName := filepath.Base(eventsFilePath)
	writtenEventsFilePath := filepath.Join(p.writeDir, utils.WidgetsDir, utils.AnalyticsWidgetFolder, "events",
		eventsFileName)
	if err := ValidateEventsFile(writtenEventsFilePath); err != nil {
		return fmt.Errorf("checking the events file: %w", err)
	}
	return utils.CopyFile(writtenEventsFilePath, eventsFilePath)
}

// EventsFilePath returns the path of the events file of the replay file replayFilePath: <replay name>_events.json
// next to it, where Run copies the file the analytics widget wrote.
func EventsFilePath(replayFilePath string) string {
	return strings.TrimSuffix(replayFilePath, filepath.Ext(replayFilePath)) + "_events.json"
}

// ValidateEventsFile returns an error if the events file filePath is missing, incomplete (cut off, or it does not end
// with the game over) or older than the embedded widget's format version. It reads only the format version and the
// end (events.ReadFormatVersionFile), which is much faster than reading the whole file.
func ValidateEventsFile(filePath string) error {
	file, err := events.ReadFormatVersionFile(filePath)
	if err != nil {
		return err
	}
	if file.FormatVersion < widget.FormatVersion {
		return fmt.Errorf("the events file has format version %d, the widget writes %d", file.FormatVersion,
			widget.FormatVersion)
	}
	return nil
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
	archiveCaches, err := filepath.Glob(filepath.Join(p.config.DataDir, utils.CacheDir, "ArchiveCache*.lua"))
	if err != nil {
		return err
	}

	//nolint:gosec // This directory does not contain sensitive data
	if err := os.MkdirAll(
		filepath.Join(p.writeDir, utils.CacheDir),
		0o755,
	); err != nil {
		return err
	}
	for _, archiveCache := range archiveCaches {
		if err := utils.CopyFile(
			archiveCache,
			filepath.Join(p.writeDir, utils.CacheDir, filepath.Base(archiveCache)),
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
	//nolint:gosec // This file does not contain sensitive data
	return os.WriteFile(
		filepath.Join(p.writeDir, "springsettings.cfg"),
		[]byte(strings.Join(lines, "\n")),
		0o644,
	)
}

// prepareWriteDirWidget writes the embedded analytics widget into the write dir's user widgets folder, so the playback
// loads it without it being installed in the data dir.
func (p *Player) prepareWriteDirWidget() error {
	if err := utils.WriteTemplate(
		slog.New(slog.DiscardHandler),
		widget.Template,
		filepath.Join(p.writeDir, utils.WidgetsDir, utils.AnalyticsWidgetFolder),
	); err != nil {
		return err
	}
	return nil
}

// prepareWriteDirWidgetConfig writes the widget config file into the write dir with the analytics widget and the user
// widgets of the data dir in its order list (see WidgetConfig), so games whose widget handler enables only the user
// widgets in that list run them too.
func (p *Player) prepareWriteDirWidgetConfig() error {
	nameLiterals, err := UserWidgetNameLiterals(filepath.Join(p.config.DataDir, utils.WidgetsDir))
	if err != nil {
		return fmt.Errorf("reading the names of the user widgets: %w", err)
	}

	filePath := filepath.Join(p.writeDir, utils.WidgetConfigFile)
	//nolint:gosec // This directory does not contain sensitive data
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		return err
	}
	//nolint:gosec // This file does not contain sensitive data
	return os.WriteFile(filePath, WidgetConfig(append(nameLiterals, `"`+widget.Name+`"`)), 0o644)
}

// runSpringHeadless plays the replay filePath with the engine in engineDir. The engine's output replaces <replay
// name>.log next to the replay, which the error names if the engine fails.
func (p *Player) runSpringHeadless(ctx context.Context, engineDir string, filePath string) error {
	logFilePath := strings.TrimSuffix(filePath, filepath.Ext(filePath)) + ".log"
	//nolint:gosec // This file does not contain sensitive data
	logFile, err := os.OpenFile(
		logFilePath,
		os.O_WRONLY|os.O_CREATE|os.O_TRUNC,
		0o644,
	)
	if err != nil {
		return err
	}
	defer logFile.Close() //nolint:errcheck

	//nolint:gosec // The user does not control the command input.
	command := exec.CommandContext(ctx, filepath.Join(engineDir, "spring-headless"),
		// Isolation keeps the engine from adding the home and system data dirs (DataDirLocater::LocateDataDirs): it
		// uses only the write dir, the engine dir and SPRING_DATADIR.
		"--isolation",
		"--write-dir", p.writeDir,
		"--config", filepath.Join(p.writeDir, "springsettings.cfg"),
		filePath,
	)
	// The widget writes its events file relative to the working directory.
	command.Dir = p.writeDir
	// The data dir is added read only: the engine writes into its write dir only.
	command.Env = append(os.Environ(), "SPRING_DATADIR="+p.config.DataDir)
	command.Cancel = func() error {
		// Ask the engine to quit first. Windows has no SIGTERM, so the engine is killed after WaitDelay there.
		return command.Process.Signal(syscall.SIGTERM)
	}
	command.WaitDelay = 5 * time.Second
	command.Stdout = logFile
	command.Stderr = logFile

	if err := command.Run(); err != nil {
		return errors.Join(
			fmt.Errorf("running the replay (see %s): %w", logFilePath, err),
			ctx.Err(),
		)
	}
	return nil
}

func (p *Player) shortReplayPath(filePath string) string {
	return utils.RelativePath(utils.DemosDir(p.config.DataDir), filePath)
}

func (p *Player) replayDate(replayInfo *ReplayInfo) string {
	return replayInfo.StartTime.Format("2006.01.02")
}

package replayplayer

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/backbone81/bar-replay-analytics-go/internal/playback/widget"
	"github.com/backbone81/bar-replay-analytics-go/internal/utils"
)

type Player struct {
	config Config

	writeDir string
}

func New(options ...Option) (*Player, error) {
	config := DefaultConfig
	for _, option := range options {
		option(&config)
	}
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

func (p *Player) Run(ctx context.Context, filePath string) error {
	var err error
	filePath, err = filepath.Abs(filePath)
	if err != nil {
		return err
	}

	// TODO: Inspect the replay to see if the demo stream size is positive and which engine was used.
	// TODO: Remember the game duration, the map and the players.

	// TODO: Download engine in the required version if missing.

	// TODO: Download map if missing.

	logger := p.config.Logger.With("file-path", filePath)
	start := time.Now()
	logger.Info("Starting replay")
	var ingameTime time.Duration
	defer func() {
		logger.Info("Finished replay", "wall-time", time.Since(start), "ingame-time", ingameTime)
	}()

	// All files are written into a temporary write directory. That way we do not pollute the data directory.
	p.writeDir, err = os.MkdirTemp("", "bar-replay-analytics-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(p.writeDir) //nolint:errcheck

	if err := p.prepareWriteDir(); err != nil {
		return fmt.Errorf("preparing write dir: %w", err)
	}
	if err := p.runSpringHeadless(ctx, filePath); err != nil {
		return fmt.Errorf("running spring-headless: %w", err)
	}

	// TODO: Copy the events file besides the replay file.

	return nil
}

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
	return nil
}

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

func (p *Player) prepareWriteDirSpringSettings() error {
	lines := []string{
		// Setting the host port to 0 will make the engine allocate a random free port. That way we do not run into
		// port collisions when multiple replays run in parallel.
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

// runSpringHeadless plays the replay filePath. The engine's output goes into <replay>.log next to the replay, which it
// replaces, and into the error if the engine fails.
func (p *Player) runSpringHeadless(ctx context.Context, filePath string) error {
	//nolint:gosec // This file does not contain sensitive data
	logFile, err := os.OpenFile(
		strings.TrimSuffix(filePath, filepath.Ext(filePath))+".log",
		os.O_WRONLY|os.O_CREATE|os.O_TRUNC,
		0o644,
	)
	if err != nil {
		return err
	}
	defer logFile.Close() //nolint:errcheck

	var outputBuffer bytes.Buffer
	output := io.MultiWriter(&outputBuffer, logFile)
	//nolint:gosec // The user does not control the command input.
	command := exec.CommandContext(ctx, p.config.SpringHeadlessFilePath,
		"--isolation",
		"--write-dir", p.writeDir,
		"--config", filepath.Join(p.writeDir, "springsettings.cfg"),
		filePath,
	)
	command.Dir = p.writeDir
	command.Env = append(os.Environ(), "SPRING_DATADIR="+p.config.DataDir)
	command.Cancel = func() error {
		// Ask the engine to quit first. Windows has no SIGTERM, so the engine is killed after WaitDelay there.
		return command.Process.Signal(syscall.SIGTERM)
	}
	command.WaitDelay = 5 * time.Second
	command.Stdout = output
	command.Stderr = output

	if err := command.Run(); err != nil {
		return errors.Join(
			fmt.Errorf("running the replay: %w", err),
			ctx.Err(),
			errors.New(outputBuffer.String()),
		)
	}
	return nil
}

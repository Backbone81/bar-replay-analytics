package replayplayer

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type AssetManager struct {
	dataDir string
}

func NewAssetManager(dataDir string) (*AssetManager, error) {
	var err error
	dataDir, err = filepath.Abs(dataDir)
	if err != nil {
		return nil, err
	}
	return &AssetManager{
		dataDir: dataDir,
	}, nil
}

func (m *AssetManager) downloadMap(ctx context.Context, engineVersion string, name string) error {
	return m.runPrDownloader(ctx, engineVersion, "map", name)
}

func (m *AssetManager) downloadGame(ctx context.Context, engineVersion string, name string) error {
	return m.runPrDownloader(ctx, engineVersion, "game", name)
}

func (m *AssetManager) runPrDownloader(ctx context.Context, engineVersion string, kind string, name string) error {
	prDownloader := filepath.Join(m.dataDir, "engine", engineVersion, "pr-downloader")
	arguments := []string{
		"--filesystem-writepath", m.dataDir,
		"--download-" + kind, name,
	}
	prDownloaderEnvs := []string{
		// The settings of the BAR launcher (BYAR-Chobby dist_cfg/config.json): BAR's servers, not SpringRTS's.
		"PRD_HTTP_SEARCH_URL=https://files-cdn.beyondallreason.dev/find",
		"PRD_RAPID_REPO_MASTER=https://repos-cdn.beyondallreason.dev/repos.gz",
		"PRD_RAPID_USE_STREAMER=false",
	}

	var outputBuffer bytes.Buffer
	//nolint:gosec // The command parameters are not controlled by the user.
	command := exec.CommandContext(ctx, prDownloader, arguments...)
	command.Dir = m.dataDir
	command.Env = append(os.Environ(), prDownloaderEnvs...)
	command.WaitDelay = 5 * time.Second
	command.Stdout = &outputBuffer
	command.Stderr = &outputBuffer

	if err := command.Run(); err != nil {
		return errors.Join(
			fmt.Errorf("running pr-downloader: %w", err),
			ctx.Err(),
			errors.New(outputBuffer.String()),
		)
	}
	return nil
}

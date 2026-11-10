package replayplayer

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/bodgit/sevenzip"
)

type AssetManager struct {
	logger  *slog.Logger
	dataDir string
}

func NewAssetManager(logger *slog.Logger, dataDir string) (*AssetManager, error) {
	var err error
	dataDir, err = filepath.Abs(dataDir)
	if err != nil {
		return nil, err
	}
	return &AssetManager{
		logger:  logger,
		dataDir: dataDir,
	}, nil
}

func (m *AssetManager) downloadMap(ctx context.Context, engineVersion string, name string) error {
	m.logger.Info("Downloading map", "name", name)
	return m.runPrDownloader(ctx, engineVersion, "map", name)
}

func (m *AssetManager) downloadGame(ctx context.Context, engineVersion string, name string) error {
	m.logger.Info("Downloading game", "name", name)
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

func (m *AssetManager) downloadEngine(ctx context.Context, engineVersion string) error {
	m.logger.Info("Downloading engine", "version", engineVersion)
	downloadUrls := []string{
		fmt.Sprintf(
			"https://files-cdn.beyondallreason.dev/download?category=engine_%s64&springname=%s",
			runtime.GOOS,
			engineVersion,
		),
		fmt.Sprintf(
			"https://github.com/beyond-all-reason/RecoilEngine/releases/download/%s/recoil_%s_amd64-%s.7z",
			engineVersion,
			engineVersion,
			runtime.GOOS,
		),
	}
	var joinedErr error
	for _, downloadUrl := range downloadUrls {
		if err := m.downloadEngineTry(ctx, engineVersion, downloadUrl); err != nil {
			joinedErr = errors.Join(joinedErr, fmt.Errorf("downloading engine from %q: %w", downloadUrl, err))
			continue
		}
		return nil
	}
	return joinedErr
}

func (m *AssetManager) downloadEngineTry(ctx context.Context, engineVersion string, downloadUrl string) error {
	filePath := filepath.Join(m.dataDir, "engine", engineVersion+".7z")
	if err := m.downloadFile(ctx, downloadUrl, filePath); err != nil {
		return err
	}
	defer os.Remove(filePath)

	if err := m.extract7z(ctx, filePath); err != nil {
		return err
	}
	return nil
}

func (m *AssetManager) downloadFile(ctx context.Context, downloadUrl string, filePath string) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadUrl, nil)
	if err != nil {
		return err
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected HTTP status: %s", response.Status)
	}

	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	_, err = io.Copy(file, response.Body)
	return errors.Join(err, file.Close())
}

func (m *AssetManager) extract7z(ctx context.Context, filePath string) error {
	baseDir := strings.TrimSuffix(filePath, filepath.Ext(filePath))

	reader, err := sevenzip.OpenReader(filePath)
	if err != nil {
		return err
	}
	defer reader.Close()

	for _, file := range reader.File {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := m.extract7zFile(file, baseDir); err != nil {
			// If extracting goes wrong, we remove the folder we extracted the engine to. Partial extractions might
			// introduce issues.
			_ = os.RemoveAll(baseDir)
			return err
		}
	}
	return nil
}

func (m *AssetManager) extract7zFile(file *sevenzip.File, baseDir string) error {
	destination := filepath.Join(baseDir, filepath.FromSlash(file.Name))
	if file.Mode().IsDir() {
		return os.MkdirAll(destination, 0o755)
	}
	if !file.Mode().IsRegular() {
		return fmt.Errorf("unsupported file type %s", file.Mode().Type())
	}

	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}

	reader, err := file.Open()
	if err != nil {
		return err
	}
	defer reader.Close()

	writer, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, file.Mode().Perm())
	if err != nil {
		return err
	}
	_, err = io.Copy(writer, reader)
	return errors.Join(err, writer.Close())
}

package replayplayer

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/bodgit/sevenzip"
)

type AssetManager struct {
	logger  *slog.Logger
	dataDir string

	inventoryMutex sync.Mutex
	inventory      *Inventory

	downloadMutex sync.Mutex
}

func NewAssetManager(logger *slog.Logger, dataDir string) (*AssetManager, error) {
	var err error
	dataDir, err = filepath.Abs(dataDir)
	if err != nil {
		return nil, err
	}
	if logger == nil {
		return nil, errors.New("logger is required")
	}

	inventory, err := NewInventory(dataDir)
	if err != nil {
		return nil, err
	}
	return &AssetManager{
		logger:    logger,
		dataDir:   dataDir,
		inventory: inventory,
	}, nil
}

func (m *AssetManager) EnsureEngineVersion(ctx context.Context, engineVersion string) (string, error) {
	// We run the first engine check under the inventory mutex only and release that mutex immediately. This allows
	// a fast check without being blocked by downloads.
	if m.engineVersionAvailable(engineVersion) {
		return m.engineDir(engineVersion), nil
	}

	// We know that we need to download the engine, so fetch the download mutex which prevents multiple parallel
	// downloads from downloading the same thing.
	m.downloadMutex.Lock()
	defer m.downloadMutex.Unlock()

	// But we need to re-check the inventory under the download mutex again, because it might be that between releasing
	// the inventory mutex and acquiring the download mutex some other go routine already finished the download.
	if m.engineVersionAvailable(engineVersion) {
		return m.engineDir(engineVersion), nil
	}

	if err := m.downloadEngine(ctx, engineVersion); err != nil {
		return "", err
	}

	// After the download of the engine is complete, we update the inventory with the new engine version while still
	// holding the download mutex.
	m.addEngineVersion(engineVersion)
	return m.engineDir(engineVersion), nil
}

func (m *AssetManager) engineVersionAvailable(engineVersion string) bool {
	m.inventoryMutex.Lock()
	defer m.inventoryMutex.Unlock()

	return m.inventory.EngineVersionAvailable(engineVersion)
}

func (m *AssetManager) addEngineVersion(engineVersion string) {
	m.inventoryMutex.Lock()
	defer m.inventoryMutex.Unlock()

	m.inventory.AddEngineVersion(engineVersion)
}

func (m *AssetManager) EnsureMap(ctx context.Context, engineVersion string, mapName string) error {
	// We run the first map check under the inventory mutex only and release that mutex immediately. This allows
	// a fast check without being blocked by downloads.
	if m.mapAvailable(mapName) {
		return nil
	}

	// We know that we need to download the map, so fetch the download mutex which prevents multiple parallel
	// downloads from downloading the same thing.
	m.downloadMutex.Lock()
	defer m.downloadMutex.Unlock()

	// But we need to re-check the inventory under the download mutex again, because it might be that between releasing
	// the inventory mutex and acquiring the download mutex some other go routine already finished the download.
	if m.mapAvailable(mapName) {
		return nil
	}

	if err := m.downloadMap(ctx, engineVersion, mapName); err != nil {
		return err
	}

	// After the download of the map is complete, we update the inventory with the new map while still
	// holding the download mutex.
	m.addMap(mapName)
	return nil
}

func (m *AssetManager) mapAvailable(mapName string) bool {
	m.inventoryMutex.Lock()
	defer m.inventoryMutex.Unlock()

	return m.inventory.MapAvailable(mapName)
}

func (m *AssetManager) addMap(mapName string) {
	m.inventoryMutex.Lock()
	defer m.inventoryMutex.Unlock()

	m.inventory.AddMap(mapName)
}

func (m *AssetManager) EnsureGame(ctx context.Context, engineVersion string, game string) error {
	// We run the first game check under the inventory mutex only and release that mutex immediately. This allows
	// a fast check without being blocked by downloads.
	if m.gameAvailable(game) {
		return nil
	}

	// We know that we need to download the game, so fetch the download mutex which prevents multiple parallel
	// downloads from downloading the same thing.
	m.downloadMutex.Lock()
	defer m.downloadMutex.Unlock()

	// But we need to re-check the inventory under the download mutex again, because it might be that between releasing
	// the inventory mutex and acquiring the download mutex some other go routine already finished the download.
	if m.gameAvailable(game) {
		return nil
	}

	if err := m.downloadGame(ctx, engineVersion, game); err != nil {
		return err
	}

	// After the download of the game is complete, we update the inventory with the new game while still
	// holding the download mutex.
	m.addGame(game)
	return nil
}

func (m *AssetManager) gameAvailable(game string) bool {
	m.inventoryMutex.Lock()
	defer m.inventoryMutex.Unlock()

	return m.inventory.GameAvailable(game)
}

func (m *AssetManager) addGame(game string) {
	m.inventoryMutex.Lock()
	defer m.inventoryMutex.Unlock()

	m.inventory.AddGame(game)
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
	prDownloader := filepath.Join(m.engineDir(engineVersion), "pr-downloader")
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
	filePath := m.engineDir(engineVersion) + ".7z"
	if err := m.downloadFile(ctx, downloadUrl, filePath); err != nil {
		return err
	}
	defer os.Remove(filePath) //nolint:errcheck

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
	defer response.Body.Close() //nolint:errcheck

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected HTTP status: %s", response.Status)
	}

	//nolint:gosec // The caller is responsible for the file path
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		return err
	}

	//nolint:gosec // The caller is responsible for the file path
	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	_, err = io.Copy(file, response.Body)
	err = errors.Join(err, file.Close())
	if err != nil {
		_ = os.Remove(filePath)
	}
	return err
}

func (m *AssetManager) extract7z(ctx context.Context, filePath string) error {
	baseDir := strings.TrimSuffix(filePath, filepath.Ext(filePath))

	reader, err := sevenzip.OpenReader(filePath)
	if err != nil {
		return err
	}
	defer reader.Close() //nolint:errcheck

	for _, file := range reader.File {
		if err := ctx.Err(); err != nil {
			// If the context is canceled, we remove the folder we extracted the engine to. Partial extractions might
			// introduce issues.
			_ = os.RemoveAll(baseDir)
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
	destination := filepath.FromSlash(file.Name)
	if !filepath.IsLocal(destination) {
		return fmt.Errorf("%q leaves the destination folder", file.Name)
	}
	destination = filepath.Join(baseDir, destination)
	if file.Mode().IsDir() {
		//nolint:gosec // This folder does not contain sensitive data
		return os.MkdirAll(destination, 0o755)
	}
	if !file.Mode().IsRegular() {
		return fmt.Errorf("unsupported file type %s", file.Mode().Type())
	}

	//nolint:gosec // This folder does not contain sensitive data
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}

	reader, err := file.Open()
	if err != nil {
		return err
	}
	defer reader.Close() //nolint:errcheck

	// In case the destination file is already there, we delete it first for a clean overwrite with a new inode.
	if err := os.Remove(destination); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}

	//nolint:gosec // The folder is safe
	writer, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, file.Mode().Perm())
	if err != nil {
		return err
	}
	_, err = io.Copy(writer, reader)
	return errors.Join(err, writer.Close())
}

func (m *AssetManager) engineDir(engineVersion string) string {
	return filepath.Join(m.dataDir, "engine", "recoil_"+engineVersion)
}

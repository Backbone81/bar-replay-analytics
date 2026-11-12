package player

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/bodgit/sevenzip"
)

// AssetManager makes sure the engines, maps and games replays need are in the data dir, and downloads them there like
// the BAR launcher does. It is safe for concurrent use by parallel playbacks.
//
// Each Ensure method checks the inventory first, which is fast and never waits for a download. Only if the asset is
// missing does it take the download mutex, so at most one download runs at a time, and check again: another goroutine
// may have downloaded the same asset while this one waited.
type AssetManager struct {
	config AssetManagerConfig

	// engineDownloadURLTemplates are config.EngineDownloadURLTemplates, parsed.
	engineDownloadURLTemplates []*template.Template

	// inventoryMutex guards inventory, which is not safe for concurrent use.
	inventoryMutex sync.Mutex
	inventory      *Inventory

	// downloadMutex serializes the downloads. It is held for a whole download and its inventory update.
	downloadMutex sync.Mutex
}

// NewAssetManager returns an AssetManager for config, with the inventory of what is already in the data dir. It fails
// if config is invalid or an engine download URL template does not parse.
func NewAssetManager(config AssetManagerConfig) (*AssetManager, error) {
	var err error
	config.DataDir, err = filepath.Abs(config.DataDir)
	if err != nil {
		return nil, err
	}
	if err := config.Validate(); err != nil {
		return nil, err
	}

	engineDownloadURLTemplates := make([]*template.Template, 0, len(config.EngineDownloadURLTemplates))
	for _, text := range config.EngineDownloadURLTemplates {
		engineDownloadURLTemplate, err := template.New("engine download URL").Parse(text)
		if err != nil {
			return nil, fmt.Errorf("parsing engine download URL template %q: %w", text, err)
		}
		engineDownloadURLTemplates = append(engineDownloadURLTemplates, engineDownloadURLTemplate)
	}

	inventory, err := NewInventory(config.DataDir)
	if err != nil {
		return nil, err
	}
	return &AssetManager{
		config:                     config,
		engineDownloadURLTemplates: engineDownloadURLTemplates,
		inventory:                  inventory,
	}, nil
}

// EnsureEngineVersion makes sure the release build of the engine engineVersion is in the data dir and returns its
// folder. It fails if engineVersion is not a release tag.
func (m *AssetManager) EnsureEngineVersion(ctx context.Context, engineVersion string) (string, error) {
	if err := validateEngineVersion(engineVersion); err != nil {
		return "", err
	}

	// Fast path: no wait for running downloads.
	if m.engineVersionAvailable(engineVersion) {
		return m.engineDir(engineVersion), nil
	}

	m.downloadMutex.Lock()
	defer m.downloadMutex.Unlock()

	// Another goroutine may have downloaded it while this one waited for the mutex.
	if m.engineVersionAvailable(engineVersion) {
		return m.engineDir(engineVersion), nil
	}

	if err := m.downloadEngine(ctx, engineVersion); err != nil {
		return "", err
	}

	// Updated under the download mutex, so a goroutine waiting for it finds the engine on its second check.
	m.addEngineVersion(engineVersion)
	return m.engineDir(engineVersion), nil
}

// engineVersionAvailable calls Inventory.EngineVersionAvailable under the inventory mutex.
func (m *AssetManager) engineVersionAvailable(engineVersion string) bool {
	m.inventoryMutex.Lock()
	defer m.inventoryMutex.Unlock()

	return m.inventory.EngineVersionAvailable(engineVersion)
}

// addEngineVersion calls Inventory.AddEngineVersion under the inventory mutex.
func (m *AssetManager) addEngineVersion(engineVersion string) {
	m.inventoryMutex.Lock()
	defer m.inventoryMutex.Unlock()

	m.inventory.AddEngineVersion(engineVersion)
}

// EnsureMap makes sure the map mapName is in the data dir. It is downloaded with the pr-downloader of the engine
// engineVersion, which EnsureEngineVersion must have made available. It fails if engineVersion is not a release tag.
func (m *AssetManager) EnsureMap(ctx context.Context, engineVersion string, mapName string) error {
	if err := validateEngineVersion(engineVersion); err != nil {
		return err
	}

	// Fast path: no wait for running downloads.
	if m.mapAvailable(mapName) {
		return nil
	}

	m.downloadMutex.Lock()
	defer m.downloadMutex.Unlock()

	// Another goroutine may have downloaded it while this one waited for the mutex.
	if m.mapAvailable(mapName) {
		return nil
	}

	if err := m.downloadMap(ctx, engineVersion, mapName); err != nil {
		return err
	}

	// Updated under the download mutex, so a goroutine waiting for it finds the map on its second check.
	m.addMap(mapName)
	return nil
}

// mapAvailable calls Inventory.MapAvailable under the inventory mutex.
func (m *AssetManager) mapAvailable(mapName string) bool {
	m.inventoryMutex.Lock()
	defer m.inventoryMutex.Unlock()

	return m.inventory.MapAvailable(mapName)
}

// addMap calls Inventory.AddMap under the inventory mutex.
func (m *AssetManager) addMap(mapName string) {
	m.inventoryMutex.Lock()
	defer m.inventoryMutex.Unlock()

	m.inventory.AddMap(mapName)
}

// EnsureGame makes sure the rapid package of the game (a gametype such as "Beyond All Reason test-31720-0f97aa5") is
// in the data dir. It is downloaded with the pr-downloader of the engine engineVersion, which EnsureEngineVersion must
// have made available. It fails if engineVersion is not a release tag.
func (m *AssetManager) EnsureGame(ctx context.Context, engineVersion string, game string) error {
	if err := validateEngineVersion(engineVersion); err != nil {
		return err
	}

	// Fast path: no wait for running downloads.
	if m.gameAvailable(game) {
		return nil
	}

	m.downloadMutex.Lock()
	defer m.downloadMutex.Unlock()

	// Another goroutine may have downloaded it while this one waited for the mutex.
	if m.gameAvailable(game) {
		return nil
	}

	if err := m.downloadGame(ctx, engineVersion, game); err != nil {
		return err
	}

	// Updated under the download mutex, so a goroutine waiting for it finds the game on its second check.
	m.addGame(game)
	return nil
}

// gameAvailable calls Inventory.GameAvailable under the inventory mutex.
func (m *AssetManager) gameAvailable(game string) bool {
	m.inventoryMutex.Lock()
	defer m.inventoryMutex.Unlock()

	return m.inventory.GameAvailable(game)
}

// addGame calls Inventory.AddGame under the inventory mutex.
func (m *AssetManager) addGame(game string) {
	m.inventoryMutex.Lock()
	defer m.inventoryMutex.Unlock()

	m.inventory.AddGame(game)
}

func (m *AssetManager) downloadMap(ctx context.Context, engineVersion string, name string) error {
	m.config.Logger.Info("Downloading map", "name", name)
	return m.runPrDownloader(ctx, engineVersion, "map", name)
}

func (m *AssetManager) downloadGame(ctx context.Context, engineVersion string, name string) error {
	m.config.Logger.Info("Downloading game", "name", name)
	return m.runPrDownloader(ctx, engineVersion, "game", name)
}

// runPrDownloader downloads the asset name of kind (map or game) into the data dir with the pr-downloader that comes
// with the engine engineVersion, as the BAR launcher does. Its output is part of the error if it fails.
func (m *AssetManager) runPrDownloader(ctx context.Context, engineVersion string, kind string, name string) error {
	prDownloader := filepath.Join(m.engineDir(engineVersion), "pr-downloader")
	arguments := []string{
		"--filesystem-writepath", m.config.DataDir,
		"--download-" + kind, name,
	}
	prDownloaderEnvs := []string{
		// The settings of the BAR launcher (BYAR-Chobby dist_cfg/config.json): BAR's servers instead of pr-downloader's
		// SpringRTS defaults, which do not have BAR's game packages.
		"PRD_HTTP_SEARCH_URL=https://files-cdn.beyondallreason.dev/find",
		"PRD_RAPID_REPO_MASTER=https://repos-cdn.beyondallreason.dev/repos.gz",
		"PRD_RAPID_USE_STREAMER=false",
	}

	var outputBuffer bytes.Buffer
	//nolint:gosec // The command parameters are not controlled by the user.
	command := exec.CommandContext(ctx, prDownloader, arguments...)
	command.Dir = m.config.DataDir
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

// downloadEngine downloads and unpacks the release build of the engine engineVersion into its engine folder. It tries
// the URLs of AssetManagerConfig.EngineDownloadURLTemplates in order.
func (m *AssetManager) downloadEngine(ctx context.Context, engineVersion string) error {
	m.config.Logger.Info("Downloading engine", "version", engineVersion)
	data := EngineDownloadURLData{
		EngineVersion: engineVersion,
		OS:            runtime.GOOS,
	}
	var joinedErr error
	for _, engineDownloadURLTemplate := range m.engineDownloadURLTemplates {
		downloadUrl, err := m.engineDownloadURL(engineDownloadURLTemplate, data)
		if err != nil {
			joinedErr = errors.Join(joinedErr, err)
			continue
		}
		if err := m.downloadEngineTry(ctx, engineVersion, downloadUrl); err != nil {
			joinedErr = errors.Join(joinedErr, fmt.Errorf("downloading engine from %q: %w", downloadUrl, err))
			continue
		}
		return nil
	}
	return joinedErr
}

// engineDownloadURL returns the URL engineDownloadURLTemplate gives for data.
func (m *AssetManager) engineDownloadURL(engineDownloadURLTemplate *template.Template, data EngineDownloadURLData) (
	string, error,
) {
	var builder strings.Builder
	if err := engineDownloadURLTemplate.Execute(&builder, data); err != nil {
		return "", fmt.Errorf("executing engine download URL template: %w", err)
	}
	return builder.String(), nil
}

// downloadEngineTry downloads the engine archive from downloadUrl next to the engine folder, unpacks it into that
// folder and removes the archive.
func (m *AssetManager) downloadEngineTry(ctx context.Context, engineVersion string, downloadUrl string) error {
	filePath := m.engineDir(engineVersion) + ".7z"
	if err := m.DownloadFile(ctx, downloadUrl, filePath); err != nil {
		return err
	}
	defer os.Remove(filePath) //nolint:errcheck

	if err := m.Extract7z(ctx, filePath); err != nil {
		return err
	}
	return nil
}

// DownloadFile saves the response to a GET of downloadUrl as filePath. A partial file is removed on error.
func (m *AssetManager) DownloadFile(ctx context.Context, downloadUrl string, filePath string) error {
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

// Extract7z unpacks the 7z archive filePath into the folder of the same name without the extension. The archive is
// unpacked in Go, so no 7-Zip needs to be installed.
func (m *AssetManager) Extract7z(ctx context.Context, filePath string) error {
	baseDir := strings.TrimSuffix(filePath, filepath.Ext(filePath))

	reader, err := sevenzip.OpenReader(filePath)
	if err != nil {
		return err
	}
	defer reader.Close() //nolint:errcheck

	for _, file := range reader.File {
		if err := ctx.Err(); err != nil {
			// A partly unpacked engine would pass as available in the next inventory, so it is removed.
			_ = os.RemoveAll(baseDir)
			return err
		}
		if err := m.extract7zFile(file, baseDir); err != nil {
			// As above: no partly unpacked engine.
			_ = os.RemoveAll(baseDir)
			return err
		}
	}
	return nil
}

// extract7zFile unpacks the archive entry file into baseDir. It rejects entries that leave baseDir and anything but
// folders and regular files, as the archive comes from the internet.
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

	// An existing file is removed rather than overwritten: the new file gets its own inode, so a running engine binary
	// (writing it would fail with "text file busy") or a hard link to the old file is not touched.
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

// engineDir returns the folder of the engine engineVersion in the data dir, named like the BAR launcher names it.
func (m *AssetManager) engineDir(engineVersion string) string {
	return filepath.Join(m.config.DataDir, "engine", "recoil_"+engineVersion)
}

package replayplayer

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

var engineFolderRegex = regexp.MustCompile(`^recoil_\d{4}\.\d{2}\.\d{2}$`)

type Inventory struct {
	dataDir string

	engineVersions []string
}

func NewInventory(dataDir string) (*Inventory, error) {
	dataDir, err := filepath.Abs(dataDir)
	if err != nil {
		return nil, err
	}

	result := Inventory{
		dataDir: dataDir,
	}
	if err := result.Refresh(); err != nil {
		return nil, err
	}
	return &result, nil
}

func (i *Inventory) EngineVersionAvailable(engineVersion string) bool {
	index := slices.Index(i.engineVersions, engineVersion)
	return index != -1
}

func (i *Inventory) AddEngineVersion(engineVersion string) {
	if i.EngineVersionAvailable(engineVersion) {
		return
	}
	i.engineVersions = append(i.engineVersions, engineVersion)
	slices.Sort(i.engineVersions)
}

func (i *Inventory) Refresh() error {
	engineVersions, err := i.getEngineVersions()
	if err != nil {
		return err
	}
	i.engineVersions = engineVersions
	return nil
}

func (i *Inventory) getEngineVersions() ([]string, error) {
	matches, err := filepath.Glob(filepath.Join(i.dataDir, "engine", "recoil_*"))
	if err != nil {
		return nil, err
	}

	var result []string
	for _, match := range matches {
		if !engineFolderRegex.MatchString(filepath.Base(match)) {
			continue
		}
		stat, err := os.Stat(match)
		if err != nil {
			return nil, err
		}
		if !stat.IsDir() {
			continue
		}

		if (!i.fileExists(filepath.Join(match, "spring-headless")) && !i.fileExists(filepath.Join(match, "spring-headless.exe"))) ||
			(!i.fileExists(filepath.Join(match, "pr-downloader")) && !i.fileExists(filepath.Join(match, "pr-downloader.exe"))) {
			continue
		}

		engineVersion := strings.TrimPrefix(filepath.Base(match), "recoil_")
		result = append(result, engineVersion)
	}
	slices.Sort(result)
	return result, nil
}

func (i *Inventory) fileExists(filePath string) bool {
	stat, err := os.Stat(filePath)
	if err != nil {
		return false
	}
	if stat.IsDir() {
		return false
	}
	return true
}

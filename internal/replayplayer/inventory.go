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
	maps           []string
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

func (i *Inventory) MapAvailable(mapName string) bool {
	mapName = i.normalizeMapName(mapName)
	index := slices.Index(i.maps, mapName)
	return index != -1
}

func (i *Inventory) AddMap(mapName string) {
	if i.MapAvailable(mapName) {
		return
	}
	i.maps = append(i.maps, i.normalizeMapName(mapName))
	slices.Sort(i.maps)
}

func (i *Inventory) Refresh() error {
	engineVersions, err := i.getEngineVersions()
	if err != nil {
		return err
	}
	i.engineVersions = engineVersions

	maps, err := i.getMaps()
	if err != nil {
		return err
	}
	i.maps = maps

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
			continue
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

func (i *Inventory) getMaps() ([]string, error) {
	matches, err := filepath.Glob(filepath.Join(i.dataDir, "maps", "*"))
	if err != nil {
		return nil, err
	}

	var result []string
	for _, match := range matches {
		stat, err := os.Stat(match)
		if err != nil {
			continue
		}
		if stat.IsDir() {
			continue
		}

		// We normalize the name to have an easier time identifying it.
		match = filepath.Base(strings.ToLower(match))
		if !slices.Contains([]string{".sd7", ".sdz"}, filepath.Ext(match)) {
			continue
		}
		match = strings.TrimSuffix(match, filepath.Ext(match))
		result = append(result, i.normalizeMapName(match))
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

func (i *Inventory) normalizeMapName(mapName string) string {
	mapNameBytes := []byte(strings.ToLower(mapName))
	for idx := range mapNameBytes {
		if 'a' <= mapNameBytes[idx] && mapNameBytes[idx] <= 'z' ||
			'0' <= mapNameBytes[idx] && mapNameBytes[idx] <= '9' {
			// We keep lower case ascii characters and numbers
			continue
		}

		// All other characters are replaced by an underscore.
		mapNameBytes[idx] = '_'
	}
	return string(mapNameBytes)
}

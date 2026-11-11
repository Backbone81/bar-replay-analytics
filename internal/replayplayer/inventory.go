package replayplayer

import (
	"bufio"
	"compress/gzip"
	"errors"
	"io/fs"
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
	games          []string
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
	_, found := slices.BinarySearch(i.engineVersions, engineVersion)
	return found
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
	_, found := slices.BinarySearch(i.maps, mapName)
	return found
}

func (i *Inventory) AddMap(mapName string) {
	if i.MapAvailable(mapName) {
		return
	}
	i.maps = append(i.maps, i.normalizeMapName(mapName))
	slices.Sort(i.maps)
}

func (i *Inventory) GameAvailable(game string) bool {
	_, found := slices.BinarySearch(i.games, game)
	return found
}

func (i *Inventory) AddGame(game string) {
	if i.GameAvailable(game) {
		return
	}
	i.games = append(i.games, game)
	slices.Sort(i.games)
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

	games, err := i.getGames()
	if err != nil {
		return err
	}
	i.games = games

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

		if (!i.fileExists(filepath.Join(match, "spring-headless")) &&
			!i.fileExists(filepath.Join(match, "spring-headless.exe"))) ||
			(!i.fileExists(filepath.Join(match, "pr-downloader")) &&
				!i.fileExists(filepath.Join(match, "pr-downloader.exe"))) {
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

func (i *Inventory) getGames() ([]string, error) {
	availablePackages, err := i.getAvailablePackages()
	if err != nil {
		return nil, err
	}

	matches, err := filepath.Glob(filepath.Join(i.dataDir, "rapid", "*", "byar", "versions.gz"))
	if err != nil {
		return nil, err
	}

	var result []string
	for _, match := range matches {
		games, err := i.getGamesFromVersions(match, availablePackages)
		if err != nil {
			return nil, err
		}
		result = append(result, games...)
	}
	slices.Sort(result)
	result = slices.Compact(result)
	return result, nil
}

func (i *Inventory) getAvailablePackages() ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(i.dataDir, "packages"))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			// On a fresh install there might be no packages yet.
			return nil, nil
		}
		return nil, err
	}

	var result []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(strings.ToLower(entry.Name())) != ".sdp" {
			continue
		}
		result = append(result, entry.Name())
	}
	slices.Sort(result)
	return result, nil
}

func (i *Inventory) getGamesFromVersions(filePath string, availablePackages []string) ([]string, error) {
	file, err := os.Open(filePath) //nolint:gosec // The caller is responsible for the right path
	if err != nil {
		return nil, err
	}
	defer file.Close() //nolint:errcheck

	reader, err := gzip.NewReader(file)
	if err != nil {
		return nil, err
	}
	defer reader.Close() //nolint:errcheck

	var result []string
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		// tag,md5,depends,name
		fields := strings.Split(scanner.Text(), ",")
		if len(fields) < 4 {
			continue
		}
		md5 := fields[1]
		name := fields[3]
		if _, found := slices.BinarySearch(availablePackages, md5+".sdp"); !found {
			continue
		}
		result = append(result, name)
	}
	if scanner.Err() != nil {
		return nil, scanner.Err()
	}
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

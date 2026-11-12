package replayplayer

import (
	"bufio"
	"compress/gzip"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

// engineVersionRegex matches the tags of released engines, which are dates. Replays name release tags, so other
// engine folders in engine/, such as local builds, are not counted.
var engineVersionRegex = regexp.MustCompile(`^\d{4}\.\d{2}\.\d{2}$`)

// validateEngineVersion returns an error if engineVersion is not the tag of a released engine. The engine version
// comes from the replay, which is untrusted, and ends up in the engine folder's path and the download URLs.
func validateEngineVersion(engineVersion string) error {
	if !engineVersionRegex.MatchString(engineVersion) {
		return fmt.Errorf("engine version %q is not a release tag (YYYY.MM.DD)", engineVersion)
	}
	return nil
}

// Inventory lists the engines, maps and games in a data dir, so a playback can tell without asking the network or
// starting the engine whether something must be downloaded. It is not safe for concurrent use.
type Inventory struct {
	dataDir string

	// The lists are sorted for binary search.
	engineVersions []string // engine tags, e.g. 2026.07.04
	maps           []string // normalized map names (see NormalizeMapName)
	games          []string // rapid names, e.g. "Beyond All Reason test-31720-0f97aa5"
}

// NewInventory returns the inventory of the data dir dataDir.
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

// EngineVersionAvailable returns whether the engine engineVersion is in the data dir.
func (i *Inventory) EngineVersionAvailable(engineVersion string) bool {
	_, found := slices.BinarySearch(i.engineVersions, engineVersion)
	return found
}

// AddEngineVersion adds the engine engineVersion after it was downloaded, without scanning the data dir again.
func (i *Inventory) AddEngineVersion(engineVersion string) {
	if i.EngineVersionAvailable(engineVersion) {
		return
	}
	i.engineVersions = append(i.engineVersions, engineVersion)
	slices.Sort(i.engineVersions)
}

// MapAvailable returns whether the map mapName (a start script's mapname) is in the data dir.
func (i *Inventory) MapAvailable(mapName string) bool {
	mapName = NormalizeMapName(mapName)
	_, found := slices.BinarySearch(i.maps, mapName)
	return found
}

// AddMap adds the map mapName after it was downloaded, without scanning the data dir again.
func (i *Inventory) AddMap(mapName string) {
	if i.MapAvailable(mapName) {
		return
	}
	i.maps = append(i.maps, NormalizeMapName(mapName))
	slices.Sort(i.maps)
}

// GameAvailable returns whether the game (a start script's gametype) is in the data dir.
func (i *Inventory) GameAvailable(game string) bool {
	_, found := slices.BinarySearch(i.games, game)
	return found
}

// AddGame adds the game after it was downloaded, without scanning the data dir again.
func (i *Inventory) AddGame(game string) {
	if i.GameAvailable(game) {
		return
	}
	i.games = append(i.games, game)
	slices.Sort(i.games)
}

// Refresh scans the data dir again.
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

// getEngineVersions returns the tags of the released engines in engine/recoil_<tag>/. An engine counts only with both
// its spring-headless, which plays the replays, and its pr-downloader, which downloads maps and games, so a partial
// folder is downloaded again.
func (i *Inventory) getEngineVersions() ([]string, error) {
	matches, err := filepath.Glob(filepath.Join(i.dataDir, "engine", "recoil_*"))
	if err != nil {
		return nil, err
	}

	var result []string
	for _, match := range matches {
		engineVersion := strings.TrimPrefix(filepath.Base(match), "recoil_")
		if validateEngineVersion(engineVersion) != nil {
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

		result = append(result, engineVersion)
	}
	slices.Sort(result)
	return result, nil
}

// getMaps returns the normalized names of the map archives (.sd7, .sdz) in maps/.
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

		// The file name is not the map name (argent_strata_1.1.sd7 for "Argent Strata 1.1"), so both are normalized.
		match = filepath.Base(strings.ToLower(match))
		if !slices.Contains([]string{".sd7", ".sdz"}, filepath.Ext(match)) {
			continue
		}
		match = strings.TrimSuffix(match, filepath.Ext(match))
		result = append(result, NormalizeMapName(match))
	}
	slices.Sort(result)
	return result, nil
}

// getGames returns the names of the games in the rapid repos of the data dir whose package is there. A rapid repo's
// versions.gz lists every version of the game, also those never downloaded; a downloaded one has its package file in
// packages/.
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

// getAvailablePackages returns the sorted file names (<md5>.sdp) of the rapid packages in packages/.
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

// getGamesFromVersions returns the names of the games in the rapid versions file filePath whose package is in
// availablePackages.
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
		// A line is "tag,md5,depends,name"; the package file is named after the md5.
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

// fileExists returns whether filePath exists and is not a folder.
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

// NormalizeMapName returns mapName in lower case with every character but ASCII letters and digits replaced by an
// underscore. This matches a map's name to the name of its archive file, which the map's uploader chose; a map whose
// file name differs in more than that is not found, so EnsureMap runs pr-downloader for it.
func NormalizeMapName(mapName string) string {
	mapNameBytes := []byte(strings.ToLower(mapName))
	for idx := range mapNameBytes {
		if 'a' <= mapNameBytes[idx] && mapNameBytes[idx] <= 'z' ||
			'0' <= mapNameBytes[idx] && mapNameBytes[idx] <= '9' {
			continue
		}
		mapNameBytes[idx] = '_'
	}
	return string(mapNameBytes)
}

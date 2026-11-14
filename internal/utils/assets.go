package utils

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

// ValidateEngineVersion returns an error if engineVersion is not the tag of a released engine. The engine version
// comes from the replay, which is untrusted, and ends up in the engine folder's path and the download URLs.
func ValidateEngineVersion(engineVersion string) error {
	if !engineVersionRegex.MatchString(engineVersion) {
		return fmt.Errorf("engine version %q is not a release tag (YYYY.MM.DD)", engineVersion)
	}
	return nil
}

// EngineVersions returns the sorted tags of the released engines in the data dir dataDir (see EngineVersionDir). An
// engine counts only with both its spring-headless and its pr-downloader.
func EngineVersions(dataDir string) ([]string, error) {
	entries, err := os.ReadDir(EngineDir(dataDir))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			// On a fresh install there might be no engines yet.
			return nil, nil
		}
		return nil, err
	}

	var result []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		engineVersion, found := strings.CutPrefix(entry.Name(), engineVersionDirPrefix)
		if !found || ValidateEngineVersion(engineVersion) != nil {
			continue
		}

		if !ExecutableExists(SpringHeadlessFilePath(dataDir, engineVersion)) ||
			!ExecutableExists(PrDownloaderFilePath(dataDir, engineVersion)) {
			continue
		}

		result = append(result, engineVersion)
	}
	slices.Sort(result)
	return result, nil
}

// Maps returns the sorted normalized names (see NormalizeMapName) of the map archives (.sd7, .sdz) in the data dir
// dataDir (see MapsDir).
func Maps(dataDir string) ([]string, error) {
	entries, err := os.ReadDir(MapsDir(dataDir))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			// On a fresh install there might be no maps yet.
			return nil, nil
		}
		return nil, err
	}

	var result []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// The file name is not the map name (argent_strata_1.1.sd7 for "Argent Strata 1.1"), so both are normalized.
		fileName := strings.ToLower(entry.Name())
		if !slices.Contains([]string{".sd7", ".sdz"}, filepath.Ext(fileName)) {
			continue
		}

		result = append(result, NormalizeMapName(strings.TrimSuffix(fileName, filepath.Ext(fileName))))
	}
	slices.Sort(result)
	return result, nil
}

// Games returns the sorted names of the games in the rapid repos of the data dir dataDir (see RapidDir) whose package
// is there. A rapid repo's versions.gz lists every version of the game, also those never downloaded; a downloaded one
// has its package file in packages/ (see PackagesDir).
func Games(dataDir string) ([]string, error) {
	packages, err := availablePackages(dataDir)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(RapidDir(dataDir))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			// On a fresh install there might be no rapid repos yet.
			return nil, nil
		}
		return nil, err
	}

	var result []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		versionsFilePath := filepath.Join(RapidDir(dataDir), entry.Name(), "byar", "versions.gz")
		games, err := gamesFromVersions(versionsFilePath, packages)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		result = append(result, games...)
	}
	slices.Sort(result)
	return slices.Compact(result), nil
}

// availablePackages returns the sorted file names (<md5>.sdp) of the rapid packages in the data dir dataDir.
func availablePackages(dataDir string) ([]string, error) {
	entries, err := os.ReadDir(PackagesDir(dataDir))
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

// gamesFromVersions returns the names of the games in the rapid versions file filePath whose package is in
// availablePackages.
func gamesFromVersions(filePath string, availablePackages []string) ([]string, error) {
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

// NormalizeMapName returns mapName in lower case with every character but ASCII letters and digits replaced by an
// underscore. This matches a map's name to the name of its archive file, which the map's uploader chose; a map whose
// file name differs in more than that is not found, so the map is downloaded again.
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

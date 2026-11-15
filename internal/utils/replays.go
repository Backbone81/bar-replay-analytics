package utils

import (
	"os"
	"path/filepath"
	"slices"
)

// ResolveReplayPaths returns the replay files paths stand for. Relative paths are resolved relative to the data demos
// directory, absolute paths are resolved as absolute. Paths to a directory resolve to all replays in that folder.
func ResolveReplayPaths(dataDir string, paths []string) ([]string, error) {
	var result []string
	for _, path := range paths {
		resolvedPaths, err := resolveReplayPath(dataDir, path)
		if err != nil {
			return nil, err
		}
		result = append(result, resolvedPaths...)
	}
	slices.Sort(result)
	return slices.Compact(result), nil
}

// resolveReplayPath returns the absolute paths of the replay files path stands for. A relative path is relative to the
// data dir's demos/ folder, where BAR saves replays, so a bare file name works; an absolute one is used as is. A folder
// stands for the replay files directly in it.
func resolveReplayPath(dataDir string, path string) ([]string, error) {
	if !filepath.IsAbs(path) {
		path = filepath.Join(DemosDir(dataDir), path)
	}

	path, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	stat, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	if stat.IsDir() {
		return replaysInDir(path)
	}
	return []string{path}, nil
}

// replaysInDir returns the paths of the .sdfz files directly in folder, not in its subfolders, like BAR's replay
// browser lists them.
func replaysInDir(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var result []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if filepath.Ext(entry.Name()) != ".sdfz" {
			continue
		}
		result = append(result, filepath.Join(dir, entry.Name()))
	}
	return result, nil
}

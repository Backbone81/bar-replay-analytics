package playback

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

// mapArchiveExtensions are the file name extensions of map archives, in lower case.
var mapArchiveExtensions = []string{".sd7", ".sdz"}

// nonAlphanumericPattern matches a run of characters which NormalizeMapName replaces.
var nonAlphanumericPattern = regexp.MustCompile(`[^a-z0-9]+`)

// NormalizeMapName returns a map name or a map archive's file name without extension in the form in which both
// match, as they differ in case and punctuation: "Argent Strata 1.1" and "argent_strata_1.1" are "argent_strata_1_1".
func NormalizeMapName(name string) string {
	return strings.Trim(nonAlphanumericPattern.ReplaceAllString(strings.ToLower(name), "_"), "_")
}

// mapArchives returns the map archives (.sd7, .sdz) in mapsDir, sorted.
func mapArchives(mapsDir string) ([]string, error) {
	entries, err := os.ReadDir(mapsDir)
	if err != nil {
		return nil, err
	}

	var archives []string
	for _, entry := range entries {
		archive := filepath.Join(mapsDir, entry.Name())
		if slices.Contains(mapArchiveExtensions, strings.ToLower(filepath.Ext(entry.Name()))) && isFile(archive) {
			archives = append(archives, archive)
		}
	}
	return archives, nil
}

// FindMapArchive returns the archive in mapsDir whose file name matches mapName (see NormalizeMapName), and whether
// there is one. A missing mapsDir holds no map.
func FindMapArchive(mapsDir string, mapName string) (string, bool, error) {
	archives, err := mapArchives(mapsDir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "", false, nil
		}
		return "", false, err
	}
	for _, archive := range archives {
		if NormalizeMapName(strings.TrimSuffix(filepath.Base(archive), filepath.Ext(archive))) ==
			NormalizeMapName(mapName) {
			return archive, true, nil
		}
	}
	return "", false, nil
}

// findOrDownloadMap returns the map archive of mapName in the data dir's maps/, downloaded with pr-downloader if
// missing. A downloaded archive whose file name does not match the map name is renamed to the normalized map name,
// so FindMapArchive finds it next time; the engine reads the map's name from inside the archive.
func (p *Player) findOrDownloadMap(ctx context.Context, mapName string, engine string) (string, error) {
	mapsDir := filepath.Join(p.dataDir, "maps")
	if archive, found, err := FindMapArchive(mapsDir, mapName); err != nil || found {
		return archive, err
	}

	if err := os.MkdirAll(mapsDir, dirPermissions); err != nil {
		return "", err
	}

	before, err := mapArchives(mapsDir)
	if err != nil {
		return "", err
	}

	if err := p.prDownload(ctx, engine, "map", mapName); err != nil {
		return "", fmt.Errorf("map %s is not in %s and could not be downloaded: %w", mapName, mapsDir, err)
	}

	after, err := mapArchives(mapsDir)
	if err != nil {
		return "", err
	}

	var downloaded []string
	for _, archive := range after {
		if !slices.Contains(before, archive) {
			downloaded = append(downloaded, archive)
		}
	}

	if len(downloaded) != 1 {
		return "", fmt.Errorf("downloading map %s added %d archives to %s, not one", mapName, len(downloaded), mapsDir)
	}

	archive := downloaded[0]
	extension := filepath.Ext(archive)
	if NormalizeMapName(strings.TrimSuffix(filepath.Base(archive), extension)) == NormalizeMapName(mapName) {
		return archive, nil
	}
	renamed := filepath.Join(mapsDir, NormalizeMapName(mapName)+extension)
	return renamed, os.Rename(archive, renamed)
}

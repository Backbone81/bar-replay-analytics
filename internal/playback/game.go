package playback

import (
	"bufio"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FindGamePackage returns the path of the rapid package of the game gameType (the setup script's gametype, such as
// "Beyond All Reason test-31720-0f97aa5") in dataDir, and whether there is one.
//
// pr-downloader lists the packages in rapid/<host>/<repo>/versions.gz, one "tag,md5,depends,name" per line, and
// stores each one as packages/<md5>.sdp, with its files in pool/. A listed package whose .sdp is missing does not
// count.
func FindGamePackage(dataDir string, gameType string) (string, bool, error) {
	versionsFiles, err := filepath.Glob(filepath.Join(dataDir, "rapid", "*", "*", "versions.gz"))
	if err != nil {
		return "", false, err
	}

	for _, versionsFile := range versionsFiles {
		md5s, err := readRapidVersions(versionsFile, gameType)
		if err != nil {
			return "", false, fmt.Errorf("%s: %w", versionsFile, err)
		}

		for _, md5 := range md5s {
			gamePackage := filepath.Join(dataDir, "packages", md5+".sdp")
			if isFile(gamePackage) {
				return gamePackage, true, nil
			}
		}
	}
	return "", false, nil
}

// rapidVersionsFields is the number of fields of a line of a rapid versions file: tag, md5, depends and name.
const rapidVersionsFields = 4

// readRapidVersions returns the md5 of every package named gameType in the rapid versions file versionsFile.
func readRapidVersions(versionsFile string, gameType string) ([]string, error) {
	file, err := os.Open(versionsFile) //nolint:gosec // The data dir's files are what the package works with.
	if err != nil {
		return nil, err
	}
	defer file.Close() //nolint:errcheck

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return nil, err
	}
	defer gzipReader.Close() //nolint:errcheck

	var md5s []string
	scanner := bufio.NewScanner(gzipReader)
	for scanner.Scan() {
		// The name may contain commas, the other fields do not.
		fields := strings.SplitN(scanner.Text(), ",", rapidVersionsFields)
		if len(fields) == rapidVersionsFields && fields[3] == gameType && fields[1] != "" &&
			filepath.Base(fields[1]) == fields[1] {
			md5s = append(md5s, fields[1])
		}
	}
	return md5s, errors.Join(scanner.Err(), gzipReader.Close())
}

// findOrDownloadGame returns the rapid package of the game gameType (see FindGamePackage), downloaded with
// pr-downloader if missing.
func (p *Player) findOrDownloadGame(ctx context.Context, gameType string, engine string) (string, error) {
	if gamePackage, found, err := FindGamePackage(p.dataDir, gameType); err != nil || found {
		return gamePackage, err
	}
	if err := p.prDownload(ctx, engine, "game", gameType); err != nil {
		return "", fmt.Errorf("game %s is not in %s and could not be downloaded: %w", gameType, p.dataDir, err)
	}

	gamePackage, found, err := FindGamePackage(p.dataDir, gameType)
	if err != nil {
		return "", err
	}

	if !found {
		return "", fmt.Errorf("pr-downloader did not download game %s into %s", gameType, p.dataDir)
	}
	return gamePackage, nil
}

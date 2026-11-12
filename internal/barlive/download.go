package barlive

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"

	"github.com/backbone81/bar-replay-analytics-go/internal/demo"
)

// demoFileNamePattern matches the characters demo file names are made of, so a name from bar-db cannot leave the
// directory or the URL path. The engine names a demo file after the map, which may hold an apostrophe ("Devil's
// Postpiles 1.1.1").
var demoFileNamePattern = regexp.MustCompile(`^[a-zA-Z0-9._ '-]+$`)

// ValidateDemoFileName returns an error if demoFileName is not a plain file name of the characters demo files are
// named with. DownloadDemo and DownloadDemoIntoDir call it before they use a file name from bar-db in a URL or a path.
func ValidateDemoFileName(demoFileName string) error {
	// url.JoinPath and filepath.Join resolve "." and "..", which would name another path.
	if !demoFileNamePattern.MatchString(demoFileName) || demoFileName == "." || demoFileName == ".." {
		return fmt.Errorf("unexpected characters in demo file name %q", demoFileName)
	}
	return nil
}

// DownloadDemo downloads the demo file demoFileName from the storage and writes it to writer. If it fails, writer may
// have received part of the file.
func (c *Client) DownloadDemo(ctx context.Context, demoFileName string, writer io.Writer) error {
	if err := ValidateDemoFileName(demoFileName); err != nil {
		return err
	}

	requestURL, err := url.JoinPath(c.config.StorageURL, "demos", url.PathEscape(demoFileName))
	if err != nil {
		return err
	}

	if err := c.downloadDemo(ctx, requestURL, writer); err != nil {
		return fmt.Errorf("GET %s: %w", requestURL, err)
	}
	return nil
}

// downloadDemo implements DownloadDemo for the URL of the demo.
func (c *Client) downloadDemo(ctx context.Context, requestURL string, writer io.Writer) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return err
	}

	c.log.Debug("HTTP GET", "url", requestURL)
	response, err := c.httpClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close() //nolint:errcheck

	if response.StatusCode == http.StatusNotFound {
		return errors.New("demo file not found on remote server")
	}

	if response.StatusCode != http.StatusOK {
		var responseData bytes.Buffer
		_, _ = io.Copy(&responseData, response.Body)
		return fmt.Errorf("unexpected HTTP status %s: %s", response.Status, responseData.String())
	}

	_, err = io.Copy(writer, response.Body)
	return err
}

// DownloadDemoIntoDir downloads the demo file of replay, as DownloadDemo does, stores it in dir under Replay.FileName
// and returns its path. It takes the whole Replay to check the file's game ID against Replay.ID. It never overwrites a
// file: if dir already holds one of that name, it returns an error matching fs.ErrExist.
//
// The download goes to the file name with ".part" appended, which is renamed only once it is complete and its header
// names the game of replay (Replay.ID), so dir never holds a partial or wrong file under the name. The .part file is
// removed if the download fails or the context is cancelled. One left behind by a killed process stays visible next to
// the replays and is overwritten by the next download of the replay.
func (c *Client) DownloadDemoIntoDir(ctx context.Context, replay *Replay, dir string) (string, error) {
	if err := ValidateDemoFileName(replay.FileName); err != nil {
		return "", err
	}
	filePath := filepath.Join(dir, replay.FileName)
	if err := checkNotExist(filePath); err != nil {
		return "", err
	}

	partPath := filePath + ".part"
	// ValidateDemoFileName made sure that partPath stays inside dir; replays are not confidential, see filePermissions.
	file, err := os.OpenFile(partPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644) //nolint:gosec
	if err != nil {
		return "", err
	}
	renamed := false
	defer func() {
		if !renamed {
			_ = os.Remove(partPath)
		}
	}()

	// Close can report a failed write, so its error counts like that of the download.
	downloadErr := c.DownloadDemo(ctx, replay.FileName, file)
	if err := errors.Join(downloadErr, file.Close()); err != nil {
		return "", err
	}

	demoFile, err := demo.ReadFile(partPath, demo.ReadUntilHeader)
	if err != nil {
		return "", fmt.Errorf("demo file %s: %w", replay.FileName, err)
	}
	if gameID := demoFile.FileHeader.GameIDAsString(); gameID != replay.ID {
		return "", fmt.Errorf("demo file %s: holds game %s, want %s", replay.FileName, gameID, replay.ID)
	}

	if err := os.Rename(partPath, filePath); err != nil {
		return "", err
	}
	renamed = true
	return filePath, nil
}

// checkNotExist returns an error matching fs.ErrExist if filePath exists, or the error of looking it up.
func checkNotExist(filePath string) error {
	_, err := os.Lstat(filePath)
	if err == nil {
		return fmt.Errorf("%s: %w", filePath, fs.ErrExist)
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}

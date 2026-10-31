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

// demoFileNamePattern describes the characters a demo file is allowed to contain. This guards us against directory
// traversals or other security issues, by not allowing special characters in the demo file name. The engine names a
// demo file after the map, and map names may hold an apostrophe, as "Devil's Postpiles 1.1.1" does.
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

// DownloadDemoIntoDir downloads the demo file of replay, as DownloadDemo does, and stores it in dir under its
// Replay.FileName. It returns the path of the file. Use DownloadDemo to write a demo file anywhere else, and
// DownloadDemoIntoDir to store it as a file: it takes the whole Replay, as it checks the downloaded file against
// Replay.ID. It never overwrites a file: if dir already holds one of that name, it returns an error matching
// fs.ErrExist.
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
	file, err := os.OpenFile(partPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, filePermissions) //nolint:gosec
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

	header, err := demo.ReadDemoFileHeaderFromFilePath(partPath)
	if err != nil {
		return "", fmt.Errorf("demo file %s: %w", replay.FileName, err)
	}
	if gameID := header.GameIDAsString(); gameID != replay.ID {
		return "", fmt.Errorf("demo file %s: holds game %s, want %s", replay.FileName, gameID, replay.ID)
	}

	if err := os.Rename(partPath, filePath); err != nil {
		return "", err
	}
	renamed = true
	return filePath, nil
}

// filePermissions are the permissions of the files the package writes: replays and the replay index hold nothing
// confidential, so everyone may read them.
const filePermissions = 0o644

// dirPermissions are the permissions of the directories the package creates, so everyone may read their files.
const dirPermissions = 0o755

// checkNotExist returns an error matching fs.ErrExist if filePath exists, and any error of looking it up.
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

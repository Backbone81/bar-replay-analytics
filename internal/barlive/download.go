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
	"github.com/backbone81/bar-replay-analytics-go/internal/utils"
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

// DownloadDemo requests the demo file demoFileName from the storage and returns its content as it arrives. The caller
// reads and closes it. A missing demo and any HTTP status but 200 fail before content is returned; a download that
// breaks off fails while it is read.
func (c *Client) DownloadDemo(ctx context.Context, demoFileName string) (io.ReadCloser, error) {
	if err := ValidateDemoFileName(demoFileName); err != nil {
		return nil, err
	}

	requestURL, err := url.JoinPath(c.config.StorageURL, "demos", url.PathEscape(demoFileName))
	if err != nil {
		return nil, err
	}

	body, err := c.downloadDemo(ctx, requestURL)
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", requestURL, err)
	}
	return body, nil
}

// downloadDemo implements DownloadDemo for the URL of the demo.
func (c *Client) downloadDemo(ctx context.Context, requestURL string) (io.ReadCloser, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, err
	}

	c.log.Debug("HTTP GET", "url", requestURL)
	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, err
	}

	if response.StatusCode == http.StatusNotFound {
		return nil, errors.Join(errors.New("demo file not found on remote server"), response.Body.Close())
	}

	if response.StatusCode != http.StatusOK {
		var responseData bytes.Buffer
		_, _ = io.Copy(&responseData, response.Body)
		return nil, errors.Join(
			fmt.Errorf("unexpected HTTP status %s: %s", response.Status, responseData.String()),
			response.Body.Close(),
		)
	}

	return response.Body, nil
}

// DownloadDemoIntoDir downloads the demo file of replay (see DownloadDemo), stores it in dir under Replay.FileName and
// returns its path. It takes the whole Replay to check the file's game ID against Replay.ID. It never overwrites a
// file: if dir already holds one of that name, it returns an error matching fs.ErrExist.
//
// The download is written with utils.WriteFileFrom, so dir never holds a partial file under the name. Then the header
// of the stored file must name the game of replay (Replay.ID); a file of another game, or no demo at all, is removed
// again. A process killed between storing and checking leaves the file unchecked.
func (c *Client) DownloadDemoIntoDir(ctx context.Context, replay *Replay, dir string) (string, error) {
	if err := ValidateDemoFileName(replay.FileName); err != nil {
		return "", err
	}
	filePath := filepath.Join(dir, replay.FileName)
	if err := checkNotExist(filePath); err != nil {
		return "", err
	}

	if err := c.downloadDemoIntoFile(ctx, replay.FileName, filePath); err != nil {
		return "", err
	}

	if err := checkDemoGameID(filePath, replay.ID); err != nil {
		return "", errors.Join(
			fmt.Errorf("demo file %s: %w", replay.FileName, err),
			os.Remove(filePath),
		)
	}
	return filePath, nil
}

// downloadDemoIntoFile downloads the demo file demoFileName (see DownloadDemo) and writes it to filePath, readable by
// everyone, as replays are not confidential (see utils.WriteFileFrom).
func (c *Client) downloadDemoIntoFile(ctx context.Context, demoFileName string, filePath string) (err error) {
	body, err := c.DownloadDemo(ctx, demoFileName)
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Join(err, body.Close())
	}()

	return utils.WriteFileFrom(filePath, body, 0o644)
}

// checkDemoGameID returns an error if the file filePath is no demo file or its header does not name the game gameID.
func checkDemoGameID(filePath string, gameID string) error {
	demoFile, err := demo.ReadFile(filePath, demo.ReadUntilHeader)
	if err != nil {
		return err
	}
	if fileGameID := demoFile.FileHeader.GameIDAsString(); fileGameID != gameID {
		return fmt.Errorf("holds game %s, want %s", fileGameID, gameID)
	}
	return nil
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

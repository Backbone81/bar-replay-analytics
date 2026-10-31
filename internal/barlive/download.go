package barlive

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
)

// demoFileNamePattern describes the characters a demo file is allowed to contain. This guards us against directory
// traversals or other security issues, by not allowing special characters in the demo file name.
var demoFileNamePattern = regexp.MustCompile(`^[a-zA-Z0-9._ -]+$`)

// DownloadDemo downloads the demo into a temporary file and returns the name of the temporary file.
func (c *Client) DownloadDemo(ctx context.Context, demoFileName string) (string, error) {
	if !demoFileNamePattern.MatchString(demoFileName) {
		return "", errors.New("unexpected characters in demo file name")
	}

	requestURL, err := url.JoinPath(c.config.StorageURL, "demos", url.PathEscape(demoFileName))
	if err != nil {
		return "", err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return "", err
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close() //nolint:errcheck

	if response.StatusCode == http.StatusNotFound {
		return "", errors.New("demo file not found on remote server")
	}

	if response.StatusCode != http.StatusOK {
		var responseData bytes.Buffer
		_, _ = io.Copy(&responseData, response.Body)
		return "", fmt.Errorf(
			"unexpected HTTP status code %d: %s: %s",
			response.StatusCode,
			response.Status,
			responseData.String(),
		)
	}

	file, err := os.CreateTemp("", "bar-demo-")
	if err != nil {
		return "", err
	}
	defer file.Close() //nolint:errcheck

	if _, err := io.Copy(file, response.Body); err != nil {
		removeErr := os.Remove(file.Name())
		return "", errors.Join(err, removeErr)
	}

	return file.Name(), nil
}

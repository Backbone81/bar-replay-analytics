package barlive

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/time/rate"
)

// Client calls the BAR Live Services. It is safe for concurrent use.
type Client struct {
	config     Config
	httpClient *http.Client
	limiter    *rate.Limiter
}

// NewClient returns a Client configured by options.
func NewClient(options ...Option) *Client {
	config := DefaultConfig
	for _, option := range options {
		option(&config)
	}
	return &Client{
		config: config,
		httpClient: &http.Client{
			Transport: config.Transport,
		},
		limiter: rate.NewLimiter(config.MaxRequestRate, 1),
	}
}

// getJSON waits for the rate limiter, requests requestURL and decodes the JSON response into target.
func (c *Client) getJSON(ctx context.Context, requestURL string, target any) error {
	if err := c.limiter.Wait(ctx); err != nil {
		return err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return fmt.Errorf("GET %s: %w", requestURL, err)
	}

	request.Header.Set("Accept", "application/json")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close() //nolint:errcheck

	if response.StatusCode != http.StatusOK {
		var responseData bytes.Buffer
		_, _ = io.Copy(&responseData, response.Body)
		return fmt.Errorf(
			"unexpected HTTP status code %d: %s: %s",
			response.StatusCode,
			response.Status,
			responseData.String(),
		)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("GET %s: reading response: %w", requestURL, err)
	}
	if err := json.Unmarshal(body, target); err != nil {
		return fmt.Errorf("GET %s: decoding response: %w", requestURL, err)
	}
	return nil
}

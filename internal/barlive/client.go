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
//
// Client sets no timeout on its requests: pass a context with a deadline to bound them.
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
	maxRequestRate := config.MaxRequestRate
	if maxRequestRate == 0 {
		maxRequestRate = rate.Inf
	}
	return &Client{
		config: config,
		httpClient: &http.Client{
			Transport: config.Transport,
		},
		limiter: rate.NewLimiter(maxRequestRate, 1),
	}
}

// reportRequest calls Config.OnRequest, if set, with requestURL.
func (c *Client) reportRequest(requestURL string) {
	if c.config.OnRequest != nil {
		c.config.OnRequest(requestURL)
	}
}

// getJSON waits for the rate limiter, requests requestURL and decodes the JSON response into target. Its errors
// name the request.
func (c *Client) getJSON(ctx context.Context, requestURL string, target any) error {
	if err := c.doGetJSON(ctx, requestURL, target); err != nil {
		return fmt.Errorf("GET %s: %w", requestURL, err)
	}
	return nil
}

// doGetJSON implements getJSON.
func (c *Client) doGetJSON(ctx context.Context, requestURL string, target any) error {
	if err := c.limiter.Wait(ctx); err != nil {
		return err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return err
	}

	request.Header.Set("Accept", "application/json")

	c.reportRequest(requestURL)
	response, err := c.httpClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close() //nolint:errcheck

	if response.StatusCode != http.StatusOK {
		var responseData bytes.Buffer
		_, _ = io.Copy(&responseData, response.Body)
		return fmt.Errorf("unexpected HTTP status %s: %s", response.Status, responseData.String())
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}
	if err := json.Unmarshal(body, target); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}
	return nil
}

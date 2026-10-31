package barlive

import (
	"net/http"

	"golang.org/x/time/rate"
)

// Option changes the Config a Client is created with.
type Option func(config *Config)

// WithTransport sets Config.Transport.
func WithTransport(transport http.RoundTripper) Option {
	return func(config *Config) {
		config.Transport = transport
	}
}

// WithTeiserverURL sets Config.TeiserverURL.
func WithTeiserverURL(teiserverURL string) Option {
	return func(config *Config) {
		config.TeiserverURL = teiserverURL
	}
}

// WithReplayAPIURL sets Config.ReplayAPIURL.
func WithReplayAPIURL(replayAPIURL string) Option {
	return func(config *Config) {
		config.ReplayAPIURL = replayAPIURL
	}
}

// WithStorageURL sets Config.StorageURL.
func WithStorageURL(storageURL string) Option {
	return func(config *Config) {
		config.StorageURL = storageURL
	}
}

// WithMaxRequestRate sets Config.MaxRequestRate. Zero means no limit.
func WithMaxRequestRate(limit rate.Limit) Option {
	return func(config *Config) {
		config.MaxRequestRate = limit
	}
}

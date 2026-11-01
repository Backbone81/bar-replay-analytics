package playback

import (
	"net/http"
	"time"
)

// Option changes the Config a Player is created with.
type Option func(config *Config)

// WithTransport sets Config.Transport.
func WithTransport(transport http.RoundTripper) Option {
	return func(config *Config) {
		config.Transport = transport
	}
}

// WithEngineCDNURL sets Config.EngineCDNURL.
func WithEngineCDNURL(engineCDNURL string) Option {
	return func(config *Config) {
		config.EngineCDNURL = engineCDNURL
	}
}

// WithEngineReleaseURL sets Config.EngineReleaseURL.
func WithEngineReleaseURL(engineReleaseURL string) Option {
	return func(config *Config) {
		config.EngineReleaseURL = engineReleaseURL
	}
}

// WithSevenZip sets Config.SevenZip.
func WithSevenZip(sevenZip string) Option {
	return func(config *Config) {
		config.SevenZip = sevenZip
	}
}

// WithStallTimeout sets Config.StallTimeout.
func WithStallTimeout(stallTimeout time.Duration) Option {
	return func(config *Config) {
		config.StallTimeout = stallTimeout
	}
}

// WithStopTimeout sets Config.StopTimeout.
func WithStopTimeout(stopTimeout time.Duration) Option {
	return func(config *Config) {
		config.StopTimeout = stopTimeout
	}
}

// WithProgressInterval sets Config.ProgressInterval.
func WithProgressInterval(progressInterval time.Duration) Option {
	return func(config *Config) {
		config.ProgressInterval = progressInterval
	}
}

// WithOnProgress sets Config.OnProgress.
func WithOnProgress(onProgress func(progress Progress)) Option {
	return func(config *Config) {
		config.OnProgress = onProgress
	}
}

// WithOnDownload sets Config.OnDownload.
func WithOnDownload(onDownload func(description string)) Option {
	return func(config *Config) {
		config.OnDownload = onDownload
	}
}

// WithWorkerThreadCount sets Config.WorkerThreadCount.
func WithWorkerThreadCount(workerThreadCount int) Option {
	return func(config *Config) {
		config.WorkerThreadCount = workerThreadCount
	}
}

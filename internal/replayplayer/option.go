package replayplayer

import "log/slog"

// Option changes a field of the Config that New and NewBatchPlayer start from (DefaultConfig).
type Option func(config *Config)

// WithLogger sets Config.Logger.
func WithLogger(logger *slog.Logger) Option {
	return func(config *Config) {
		config.Logger = logger
	}
}

// WithDataDir sets Config.DataDir.
func WithDataDir(dataDir string) Option {
	return func(config *Config) {
		config.DataDir = dataDir
	}
}

// WithWorkerThreadCount sets Config.WorkerThreadCount.
func WithWorkerThreadCount(workerThreadCount int) Option {
	return func(config *Config) {
		config.WorkerThreadCount = workerThreadCount
	}
}

// WithAssetManager sets Config.AssetManager.
func WithAssetManager(assetManager *AssetManager) Option {
	return func(config *Config) {
		config.AssetManager = assetManager
	}
}

// WithConcurrentReplays sets Config.ConcurrentReplays.
func WithConcurrentReplays(concurrentReplays int) Option {
	return func(config *Config) {
		config.ConcurrentReplays = concurrentReplays
	}
}

// WithConfig replaces the whole Config. BatchPlayer uses it to give each of its Players its own Config.
func WithConfig(newConfig Config) Option {
	return func(config *Config) {
		*config = newConfig
	}
}

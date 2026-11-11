package replayplayer

import "log/slog"

type Option func(config *Config)

func WithLogger(logger *slog.Logger) Option {
	return func(config *Config) {
		config.Logger = logger
	}
}

func WithDataDir(dataDir string) Option {
	return func(config *Config) {
		config.DataDir = dataDir
	}
}

func WithWorkerThreadCount(workerThreadCount int) Option {
	return func(config *Config) {
		config.WorkerThreadCount = workerThreadCount
	}
}

func WithAssetManager(assetManager *AssetManager) Option {
	return func(config *Config) {
		config.AssetManager = assetManager
	}
}

func WithConcurrentReplays(concurrentReplays int) Option {
	return func(config *Config) {
		config.ConcurrentReplays = concurrentReplays
	}
}

func WithConfig(newConfig Config) Option {
	return func(config *Config) {
		*config = newConfig
	}
}

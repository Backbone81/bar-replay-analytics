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

func WithSpringHeadless(filePath string) Option {
	return func(config *Config) {
		config.SpringHeadlessFilePath = filePath
	}
}

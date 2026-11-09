package replayplayer

import "log/slog"

type Option func(config *Config)

func WithLogger(logger *slog.Logger) Option {
	return func(config *Config) {
		config.Logger = logger
	}
}

func WithDir(dir string) Option {
	return func(config *Config) {
		config.DataDir = dir
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

package replayplayer

import (
	"errors"
	"log/slog"
)

type Config struct {
	Logger *slog.Logger

	DataDir           string
	WorkerThreadCount int
	AssetManager      *AssetManager
	ConcurrentReplays int
}

var DefaultConfig = Config{
	Logger: slog.New(slog.DiscardHandler),

	DataDir:           "data",
	WorkerThreadCount: 2,
	ConcurrentReplays: 1,
}

func (c Config) Validate() error {
	if c.Logger == nil {
		return errors.New("the logger is required")
	}
	if c.WorkerThreadCount < 0 {
		return errors.New("the worker thread count must not be negative")
	}
	if c.AssetManager == nil {
		return errors.New("the asset manager is required")
	}
	if c.ConcurrentReplays < 1 {
		return errors.New("concurrent replays must be at least 1")
	}

	return nil
}

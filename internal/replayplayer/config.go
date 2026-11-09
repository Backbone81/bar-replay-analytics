package replayplayer

import (
	"errors"
	"log/slog"
)

type Config struct {
	Logger *slog.Logger

	DataDir                string
	WorkerThreadCount      int
	SpringHeadlessFilePath string
}

var DefaultConfig = Config{
	Logger: slog.New(slog.DiscardHandler),

	DataDir:           "data",
	WorkerThreadCount: 2,
}

func (c Config) Validate() error {
	if c.Logger == nil {
		return errors.New("the logger is required")
	}
	if c.WorkerThreadCount < 0 {
		return errors.New("the worker thread count must not be negative")
	}
	return nil
}

package downloader

import (
	"errors"
	"log/slog"

	"github.com/backbone81/bar-replay-analytics/internal/barlive"
	"github.com/backbone81/bar-replay-analytics/internal/replay/index"
)

type Config struct {
	Logger *slog.Logger

	Dir      string
	Players  int
	Days     int
	Maps     []string
	GameType barlive.GameType

	Client *barlive.Client
	Index  *index.ReplayIndex

	DryRun bool
}

var DefaultConfig = Config{
	Logger: slog.New(slog.DiscardHandler),

	Dir:     "demos/leaderboard",
	Players: 10,
	Days:    7,

	DryRun: false,
}

func (c Config) Validate() error {
	if c.Logger == nil {
		return errors.New("the logger is required")
	}

	if c.Players < 1 {
		return errors.New("the player count must be bigger than 0")
	}
	if c.Days < 1 {
		return errors.New("the day count must be bigger than 0")
	}
	if c.GameType == "" {
		return errors.New("the game type is required")
	}
	if c.Client == nil {
		return errors.New("the bar live client is required")
	}
	if c.Index == nil {
		return errors.New("the replay index is required")
	}
	return nil
}

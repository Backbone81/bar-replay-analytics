package downloader

import (
	"log/slog"

	"github.com/backbone81/bar-replay-analytics/internal/barlive"
	"github.com/backbone81/bar-replay-analytics/internal/replay/index"
)

type Option func(config *Config)

func WithLogger(logger *slog.Logger) Option {
	return func(config *Config) {
		config.Logger = logger
	}
}

func WithDir(dir string) Option {
	return func(config *Config) {
		config.Dir = dir
	}
}

func WithPlayers(players int) Option {
	return func(config *Config) {
		config.Players = players
	}
}

func WithDays(days int) Option {
	return func(config *Config) {
		config.Days = days
	}
}

func WithMap(mapName string) Option {
	return func(config *Config) {
		config.Maps = append(config.Maps, mapName)
	}
}

func WithGameType(gameType barlive.GameType) Option {
	return func(config *Config) {
		config.GameType = gameType
	}
}

func WithClient(client *barlive.Client) Option {
	return func(config *Config) {
		config.Client = client
	}
}

func WithIndex(index *index.ReplayIndex) Option {
	return func(config *Config) {
		config.Index = index
	}
}

func WithDryRun() Option {
	return func(config *Config) {
		config.DryRun = true
	}
}

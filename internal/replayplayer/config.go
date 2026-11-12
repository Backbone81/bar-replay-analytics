package replayplayer

import (
	"errors"
	"log/slog"
)

// Config configures a Player and a BatchPlayer. Set it with the Option functions.
type Config struct {
	// Logger receives the progress and the errors of the playbacks.
	Logger *slog.Logger

	// DataDir is the BAR data dir the engine reads engines, maps and games from (passed as SPRING_DATADIR).
	DataDir string

	// WorkerThreadCount is the engine's WorkerThreadCount setting: the threads of its thread pool, the main thread
	// included. Zero leaves the engine's default of one thread per logical CPU, which is slower for a replay and
	// starves parallel playbacks.
	WorkerThreadCount int

	// AssetManager downloads missing engines, maps and games. It is required and should be shared by all Players of
	// one data dir, so parallel playbacks do not download the same asset at once.
	AssetManager *AssetManager

	// ConcurrentReplays is how many replays a BatchPlayer plays at once. NewBatchPlayer replaces zero by a value from
	// the CPUs and the available memory.
	ConcurrentReplays int

	// Filter selects the replays a BatchPlayer plays back. The zero Filter selects all.
	Filter Filter

	// MaxReplays is how many replays a BatchPlayer plays back at most: the first ones, in file path order, that match
	// Filter and whose events file is not current. A replay whose playback fails counts too. Zero means no limit.
	MaxReplays int
}

// DefaultConfig is the Config the Options are applied to. AssetManager has no default, so it must be set.
var DefaultConfig = Config{
	Logger: slog.New(slog.DiscardHandler),

	DataDir:           "data",
	WorkerThreadCount: 2,
	ConcurrentReplays: 1,
}

// Validate returns an error if the Config cannot be used.
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
	if c.MaxReplays < 0 {
		return errors.New("the maximum number of replays must not be negative")
	}

	return nil
}

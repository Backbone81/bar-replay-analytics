package replayplayer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/backbone81/bar-replay-analytics-go/internal/events"
	"github.com/backbone81/bar-replay-analytics-go/internal/playback/widget"
	"github.com/backbone81/bar-replay-analytics-go/internal/utils"
)

type BatchPlayer struct {
	config Config

	errCounter atomic.Int64
}

func NewBatchPlayer(options ...Option) (*BatchPlayer, error) {
	config := DefaultConfig
	for _, option := range options {
		option(&config)
	}
	if config.ConcurrentReplays == 0 {
		// Each replay requires about 1.5 cpu cores and 6 GB of RAM
		byCores := max(1, runtime.NumCPU()/2)
		byMemory := max(1, utils.MemoryAvailable()/(6*1024*1024*1024))
		config.ConcurrentReplays = min(byCores, byMemory)
	}
	if err := config.Validate(); err != nil {
		return nil, err
	}
	return &BatchPlayer{
		config: config,
	}, nil
}

func (p *BatchPlayer) Run(ctx context.Context, paths ...string) error {
	p.errCounter.Store(0)

	paths, err := p.resolvePaths(paths)
	if err != nil {
		return err
	}

	// Create players first. This avoids go routines erroring out and not consuming any paths.
	players := make([]*Player, p.config.ConcurrentReplays)
	for i := range p.config.ConcurrentReplays {
		player, err := New(WithConfig(p.config))
		if err != nil {
			return err
		}
		players[i] = player
	}

	// The replay file paths are communicated through this channel to the go routines executing it.
	replayFilePathsChan := make(chan string)

	var wg sync.WaitGroup
	for i := range p.config.ConcurrentReplays {
		wg.Go(func() {
			p.runReplays(ctx, players[i], replayFilePathsChan)
		})
	}

	p.feedReplayPaths(ctx, paths, replayFilePathsChan)

	close(replayFilePathsChan)
	wg.Wait()

	if ctx.Err() != nil {
		return ctx.Err()
	}
	if p.errCounter.Load() != 0 {
		return fmt.Errorf("%d errors occurred during replays", p.errCounter.Load())
	}
	return nil
}

func (p *BatchPlayer) feedReplayPaths(ctx context.Context, paths []string, replayFilePathsChan chan<- string) {
	for _, path := range paths {
		select {
		case replayFilePathsChan <- path:
		case <-ctx.Done():
			return
		}
	}
}

func (p *BatchPlayer) runReplays(ctx context.Context, player *Player, replayFilePathsChan <-chan string) {
	for replayFilePath := range replayFilePathsChan {
		if ctx.Err() != nil {
			return
		}

		if p.isEventsFileCurrent(replayFilePath) {
			p.config.Logger.Info("Skipping replay because its events file is up to date", "file-path", replayFilePath)
			continue
		}

		// TODO: introduce a player.Prepare which does the downloading, that way we can set a dedicated timeout
		// to the replay.

		if err := p.runReplay(ctx, player, replayFilePath); err != nil {
			p.config.Logger.Error("Replay failed", "file-path", replayFilePath, "err", err)
			p.errCounter.Add(1)
		}
	}
}

// runReplay plays the replay file filePath back with player, unless its events file is current (see
// isEventsFileCurrent).
func (p *BatchPlayer) runReplay(ctx context.Context, player *Player, filePath string) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	if err := player.Run(ctx, filePath); err != nil {
		return err
	}
	return nil
}

// resolvePaths returns the replay files paths name, sorted and deduplicated.
func (p *BatchPlayer) resolvePaths(paths []string) ([]string, error) {
	var result []string
	for _, path := range paths {
		resolvedPaths, err := p.resolvePath(path)
		if err != nil {
			return nil, err
		}
		result = append(result, resolvedPaths...)
	}
	slices.Sort(result)
	return slices.Compact(result), nil
}

// resolvePath returns the absolute paths of the replay file path name. A relative path is relative to the demos/
// folder of the data dir, an absolute one is used as is. A folder stands for the replay files in it.
func (p *BatchPlayer) resolvePath(path string) ([]string, error) {
	if !filepath.IsAbs(path) {
		path = filepath.Join(p.config.DataDir, "demos", path)
	}

	path, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	stat, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if stat.IsDir() {
		return p.replayFilesInFolder(path)
	}
	return []string{path}, nil
}

// replayFilesInFolder returns the paths of the .sdfz files directly in folder (not in its subfolders).
func (p *BatchPlayer) replayFilesInFolder(folder string) ([]string, error) {
	entries, err := os.ReadDir(folder)
	if err != nil {
		return nil, err
	}
	var result []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) != ".sdfz" {
			continue
		}
		result = append(result, filepath.Join(folder, entry.Name()))
	}
	return result, nil
}

// isEventsFileCurrent returns whether the replay file replayFilePath has an events file next to it. It also verifies
// that it has at least the format version of the embedded widget. Playing the replay again would therefore write the
// same events.
func (p *BatchPlayer) isEventsFileCurrent(replayFilePath string) bool {
	eventsFilePath := strings.TrimSuffix(replayFilePath, filepath.Ext(replayFilePath)) + "_events.json"
	file, err := events.ReadFile(eventsFilePath)
	if err != nil {
		// Whatever the reason is why we cannot read the events file, we report as not being current to write it again.
		return false
	}
	return file.FormatVersion >= widget.FormatVersion
}

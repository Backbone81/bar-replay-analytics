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

// BatchPlayer plays many replays back, Config.ConcurrentReplays at a time, each with a Player of its own.
type BatchPlayer struct {
	config Config

	// errCounter counts the replays of the current Run that failed. A failed replay does not stop the others.
	errCounter atomic.Int64
}

// NewBatchPlayer returns a BatchPlayer for DefaultConfig changed by options. A ConcurrentReplays of zero is replaced by
// as many playbacks as the CPUs and the available memory allow.
func NewBatchPlayer(options ...Option) (*BatchPlayer, error) {
	config := DefaultConfig
	for _, option := range options {
		option(&config)
	}
	if config.ConcurrentReplays == 0 {
		// With WorkerThreadCount 2, a 27-minute 1v1 used 1.4 to 1.7 cores and 4.8 GB; team games and larger maps need
		// more, hence 2 cores and 6 GB per playback.
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

// Run plays the replays paths back (see resolvePath), skipping those whose events file is current. It returns ctx's
// error if ctx was canceled, otherwise an error if any replay failed.
func (p *BatchPlayer) Run(ctx context.Context, paths ...string) error {
	p.errCounter.Store(0)

	paths, err := p.resolvePaths(paths)
	if err != nil {
		return err
	}

	// The Players are created before the goroutines start, so an error in New returns here instead of leaving fewer
	// goroutines to read the channel.
	players := make([]*Player, p.config.ConcurrentReplays)
	for i := range p.config.ConcurrentReplays {
		player, err := New(WithConfig(p.config))
		if err != nil {
			return err
		}
		players[i] = player
	}

	// The goroutines take the replays from this unbuffered channel, so each one starts the next replay when it is free.
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

// feedReplayPaths sends paths to replayFilePathsChan until all are taken or ctx is canceled.
func (p *BatchPlayer) feedReplayPaths(ctx context.Context, paths []string, replayFilePathsChan chan<- string) {
	for _, path := range paths {
		select {
		case replayFilePathsChan <- path:
		case <-ctx.Done():
			return
		}
	}
}

// runReplays plays the replays it receives on replayFilePathsChan back with player until the channel is closed or ctx
// is canceled. Errors are logged and counted, so one broken replay does not stop the batch.
func (p *BatchPlayer) runReplays(ctx context.Context, player *Player, replayFilePathsChan <-chan string) {
	for replayFilePath := range replayFilePathsChan {
		if ctx.Err() != nil {
			return
		}

		if p.isEventsFileCurrent(replayFilePath) {
			p.config.Logger.Info("Skipping replay because its events file is up to date", "file-path", replayFilePath)
			continue
		}

		replayInfo, err := p.prepareReplay(ctx, player, replayFilePath)
		if err != nil {
			p.config.Logger.Error("Preparing replay failed", "file-path", replayFilePath, "err", err)
			p.errCounter.Add(1)
			continue
		}
		if err := p.runReplay(ctx, player, replayInfo); err != nil {
			p.config.Logger.Error("Replay failed", "file-path", replayFilePath, "err", err)
			p.errCounter.Add(1)
			continue
		}
	}
}

// prepareReplay prepares the replay filePath with player. The timeout covers the downloads of a new engine, map and
// game, so a stalled download does not block the batch.
func (p *BatchPlayer) prepareReplay(ctx context.Context, player *Player, filePath string) (*ReplayInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Minute)
	defer cancel()

	replayInfo, err := player.Prepare(ctx, filePath)
	if err != nil {
		return nil, err
	}
	return replayInfo, nil
}

// runReplay plays the replay replayInfo describes back with player. The timeout covers the playback only (the downloads
// are done by prepareReplay), so it stops an engine that hangs.
func (p *BatchPlayer) runReplay(ctx context.Context, player *Player, replayInfo *ReplayInfo) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	if err := player.Run(ctx, replayInfo); err != nil {
		return err
	}
	return nil
}

// resolvePaths returns the replay files paths stand for (see resolvePath), sorted and deduplicated, so a replay named
// twice, e.g. by its file and its folder, is played once.
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

// resolvePath returns the absolute paths of the replay files path stands for. A relative path is relative to the data
// dir's demos/ folder, where BAR saves replays, so a bare file name works; an absolute one is used as is. A folder
// stands for the replay files directly in it.
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

// replayFilesInFolder returns the paths of the .sdfz files directly in folder, not in its subfolders, like BAR's replay
// browser lists them.
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

// isEventsFileCurrent returns whether the replay replayFilePath has a readable events file next to it with at least
// the embedded widget's format version. Playing such a replay again would write the same events.
func (p *BatchPlayer) isEventsFileCurrent(replayFilePath string) bool {
	eventsFilePath := strings.TrimSuffix(replayFilePath, filepath.Ext(replayFilePath)) + "_events.json"
	file, err := events.ReadFile(eventsFilePath)
	if err != nil {
		// A missing, incomplete or unreadable events file is written again.
		return false
	}
	return file.FormatVersion >= widget.FormatVersion
}

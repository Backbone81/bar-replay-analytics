package player

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/backbone81/bar-replay-analytics-go/internal/utils"
)

// BatchPlayer plays many replays back, Config.ConcurrentReplays at a time, each with a Player of its own.
type BatchPlayer struct {
	config Config

	// playedCounter counts the replays of the current Run that were played back.
	playedCounter atomic.Int64

	// skippedCounter counts the replays of the current Run that were not played because their events file is current.
	skippedCounter atomic.Int64

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
	// The replay paths are absolute, so the data dir must be too, for Config.LogPath.
	var err error
	config.DataDir, err = filepath.Abs(config.DataDir)
	if err != nil {
		return nil, err
	}
	if config.ConcurrentReplays == 0 {
		// With WorkerThreadCount 2, a 27-minute 1v1 used 1.4 to 1.7 cores and 4.8 GB; team games and larger maps need
		// more, hence 2 cores and 6 GB per playback.
		byCores := max(1, runtime.NumCPU()/2)
		byMemory := max(1, utils.MemoryAvailable()/(6*1024*1024*1024))
		config.ConcurrentReplays = min(byCores, byMemory)
	}
	if config.AssetManager == nil {
		assetManagerConfig := DefaultAssetManagerConfig
		assetManagerConfig.Logger = config.Logger
		assetManagerConfig.DataDir = config.DataDir
		assetManager, err := NewAssetManager(assetManagerConfig)
		if err != nil {
			return nil, err
		}
		config.AssetManager = assetManager
	}
	if err := config.Validate(); err != nil {
		return nil, err
	}
	return &BatchPlayer{
		config: config,
	}, nil
}

// Run plays the replays paths back (see resolvePath), skipping those whose events file is current and those
// Config.Filter does not match, at most Config.MaxReplays of them. It returns ctx's error if ctx was canceled,
// otherwise an error if any replay failed.
func (p *BatchPlayer) Run(ctx context.Context, paths ...string) error {
	p.config.Logger.Info("Running replays concurrently", "count", p.config.ConcurrentReplays)

	start := time.Now()
	p.playedCounter.Store(0)
	p.skippedCounter.Store(0)
	p.errCounter.Store(0)

	paths, err := p.ResolvePaths(paths)
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

	// The goroutines take the replays from this unbuffered channel, so each one starts the next replay when it is free,
	// and the replays are selected while the others play.
	replayInfosChan := make(chan *ReplayInfo)

	var wg sync.WaitGroup
	for i := range p.config.ConcurrentReplays {
		wg.Go(func() {
			p.runReplays(ctx, players[i], replayInfosChan)
		})
	}

	p.feedReplays(ctx, paths, replayInfosChan)

	close(replayInfosChan)
	wg.Wait()

	total := p.playedCounter.Load() + p.skippedCounter.Load() + p.errCounter.Load()
	played := p.playedCounter.Load() + p.errCounter.Load()
	duration := time.Since(start).Round(time.Second)
	var replayRate int
	if played > 0 {
		replayRate = int(float64(played) / duration.Hours())
	}
	p.config.Logger.Info("Finished all replays",
		"total", total,
		"played", p.playedCounter.Load(),
		"skipped", p.skippedCounter.Load(),
		"failed", p.errCounter.Load(),
		"duration", duration,
		"replays-per-hour", replayRate,
	)

	if ctx.Err() != nil {
		return ctx.Err()
	}
	if p.errCounter.Load() != 0 {
		return fmt.Errorf("%d errors occurred during replays", p.errCounter.Load())
	}
	return nil
}

// feedReplays sends the replays of paths that are selected (see selectReplay) to replayInfosChan, in the order of
// paths, until all are sent, Config.MaxReplays are sent or ctx is canceled. Replays that cannot be read are logged and
// counted as errors.
func (p *BatchPlayer) feedReplays(ctx context.Context, paths []string, replayInfosChan chan<- *ReplayInfo) {
	var sent int
	for _, path := range paths {
		if p.config.MaxReplays > 0 && sent >= p.config.MaxReplays {
			p.config.Logger.Info("Maximum number of replays is reached", "count", p.config.MaxReplays)
			return
		}
		if ctx.Err() != nil {
			return
		}

		replayInfo, selected, err := p.selectReplay(path)
		if err != nil {
			p.config.Logger.Error("Reading replay failed",
				"file-path", p.shortReplayPath(path),
				"err", err,
			)
			p.errCounter.Add(1)
			continue
		}
		if !selected {
			continue
		}

		select {
		case replayInfosChan <- replayInfo:
			sent++
		case <-ctx.Done():
			return
		}
	}
}

// selectReplay returns the ReplayInfo of the replay filePath and whether it is to be played back: its events file is
// not current and Config.Filter matches it. It only reads the replay's header and start script, so nothing is
// downloaded for a skipped replay.
func (p *BatchPlayer) selectReplay(filePath string) (*ReplayInfo, bool, error) {
	replayInfo, err := ReadReplayInfo(filePath)
	if err != nil {
		return nil, false, err
	}

	if !p.config.Filter.Matches(replayInfo) {
		return replayInfo, false, nil
	}

	if p.IsEventsFileCurrent(filePath) {
		p.config.Logger.Info("Skipping replay. Events file up to date.",
			"file-path", p.shortReplayPath(filePath),
			"date", p.replayDate(replayInfo),
		)
		p.skippedCounter.Add(1)
		return nil, false, nil
	}
	return replayInfo, true, nil
}

// runReplays plays the replays it receives on replayInfosChan back with player until the channel is closed or ctx is
// canceled. Errors are logged and counted, so one broken replay does not stop the batch.
func (p *BatchPlayer) runReplays(ctx context.Context, player *Player, replayInfosChan <-chan *ReplayInfo) {
	for replayInfo := range replayInfosChan {
		if ctx.Err() != nil {
			return
		}

		if err := p.prepareReplay(ctx, player, replayInfo); err != nil {
			p.config.Logger.Error("Preparing replay failed",
				"file-path", p.shortReplayPath(replayInfo.FilePath),
				"date", p.replayDate(replayInfo),
				"err", err,
			)
			p.errCounter.Add(1)
			continue
		}
		if err := p.runReplay(ctx, player, replayInfo); err != nil {
			p.config.Logger.Error("Replay failed",
				"file-path", p.shortReplayPath(replayInfo.FilePath),
				"date", p.replayDate(replayInfo),
				"err", err,
			)
			p.errCounter.Add(1)
			continue
		}
		p.playedCounter.Add(1)
	}
}

// prepareReplay prepares the replay replayInfo describes with player. The timeout covers the downloads of a new
// engine, map and game, so a stalled download does not block the batch.
func (p *BatchPlayer) prepareReplay(ctx context.Context, player *Player, replayInfo *ReplayInfo) error {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Minute)
	defer cancel()

	return player.Prepare(ctx, replayInfo)
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

// ResolvePaths returns the replay files paths stand for (see resolvePath), sorted and deduplicated, so a replay named
// twice, e.g. by its file and its folder, is played once.
func (p *BatchPlayer) ResolvePaths(paths []string) ([]string, error) {
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
		path = filepath.Join(utils.DemosDir(p.config.DataDir), path)
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

// IsEventsFileCurrent returns whether the replay replayFilePath has a complete events file next to it with at least
// the embedded widget's format version (see ValidateEventsFile). Playing such a replay again would write the same
// events; a missing, incomplete, unreadable or outdated one is written again.
func (p *BatchPlayer) IsEventsFileCurrent(replayFilePath string) bool {
	return ValidateEventsFile(utils.DemoEventsFilePath(replayFilePath)) == nil
}

func (p *BatchPlayer) replayDate(replayInfo *ReplayInfo) string {
	return replayInfo.StartTime.Format("2006.01.02")
}

func (p *BatchPlayer) shortReplayPath(filePath string) string {
	return utils.RelativePath(utils.DemosDir(p.config.DataDir), filePath)
}

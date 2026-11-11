package replayplayer

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

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

	// TODO: resolve the paths for relative paths, absolute paths and directories

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

		// TODO: introduce a player.Prepare which does the downloading, that way we can set a dedicated timeout
		// to the replay.

		if err := p.runReplay(ctx, player, replayFilePath); err != nil {
			p.config.Logger.Error("Replay failed", "file-path", replayFilePath, "err", err)
			p.errCounter.Add(1)
		}
	}
}

func (p *BatchPlayer) runReplay(ctx context.Context, player *Player, filePath string) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	if err := player.Run(ctx, filePath); err != nil {
		return err
	}
	return nil
}

package player_test

import (
	"log/slog"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	replayplayer "github.com/backbone81/bar-replay-analytics/internal/replay/player"
	"github.com/backbone81/bar-replay-analytics/internal/utils"
)

var _ = Describe("Option", func() {
	It("WithLogger sets the logger", func() {
		logger := slog.New(slog.DiscardHandler)
		var config replayplayer.Config

		replayplayer.WithLogger(logger)(&config)

		Expect(config.Logger).To(BeIdenticalTo(logger))
	})

	It("WithDataDir sets the data dir", func() {
		var config replayplayer.Config

		replayplayer.WithDataDir("/bar/data")(&config)

		Expect(config.DataDir).To(Equal("/bar/data"))
	})

	It("WithWorkerThreadCount sets the worker thread count", func() {
		var config replayplayer.Config

		replayplayer.WithWorkerThreadCount(3)(&config)

		Expect(config.WorkerThreadCount).To(Equal(3))
	})

	It("WithAssetManager sets the asset manager", func() {
		assetManager := newAssetManager(newDataDir())
		var config replayplayer.Config

		replayplayer.WithAssetManager(assetManager)(&config)

		Expect(config.AssetManager).To(BeIdenticalTo(assetManager))
	})

	It("WithConcurrentReplays sets the concurrent replays", func() {
		var config replayplayer.Config

		replayplayer.WithConcurrentReplays(4)(&config)

		Expect(config.ConcurrentReplays).To(Equal(4))
	})

	It("WithFilter sets the filter", func() {
		filter := utils.Filter{
			MapNames:    []string{"strata"},
			PlayerNames: []string{"Alice"},
		}
		var config replayplayer.Config

		replayplayer.WithFilter(filter)(&config)

		Expect(config.Filter).To(Equal(filter))
	})

	It("WithMaxReplays sets the maximum number of replays", func() {
		var config replayplayer.Config

		replayplayer.WithMaxReplays(5)(&config)

		Expect(config.MaxReplays).To(Equal(5))
	})

	It("WithConfig replaces the whole config", func() {
		newConfig := replayplayer.Config{DataDir: "/bar/data", ConcurrentReplays: 2}
		config := replayplayer.DefaultConfig

		replayplayer.WithConfig(newConfig)(&config)

		Expect(config).To(Equal(newConfig))
	})
})

package replayplayer_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config", func() {
	Context("Validate", func() {
		It("accepts the default config with an asset manager", func() {
			Expect(validConfig().Validate()).To(Succeed())
		})

		It("requires a logger", func() {
			config := validConfig()
			config.Logger = nil

			Expect(config.Validate()).ToNot(Succeed())
		})

		It("rejects a negative worker thread count", func() {
			config := validConfig()
			config.WorkerThreadCount = -1

			Expect(config.Validate()).ToNot(Succeed())
		})

		It("accepts a worker thread count of zero", func() {
			config := validConfig()
			config.WorkerThreadCount = 0

			Expect(config.Validate()).To(Succeed())
		})

		It("requires an asset manager", func() {
			config := validConfig()
			config.AssetManager = nil

			Expect(config.Validate()).ToNot(Succeed())
		})

		It("requires at least one concurrent replay", func() {
			config := validConfig()
			config.ConcurrentReplays = 0

			Expect(config.Validate()).ToNot(Succeed())
		})

		It("rejects a negative maximum number of replays", func() {
			config := validConfig()
			config.MaxReplays = -1

			Expect(config.Validate()).ToNot(Succeed())
		})
	})
})

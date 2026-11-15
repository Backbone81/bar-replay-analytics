package player_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	replayplayer "github.com/backbone81/bar-replay-analytics/internal/replay/player"
)

var _ = Describe("AssetManagerConfig", func() {
	Context("Validate", func() {
		It("accepts the default config", func() {
			Expect(replayplayer.DefaultAssetManagerConfig.Validate()).To(Succeed())
		})

		It("requires a logger", func() {
			config := replayplayer.DefaultAssetManagerConfig
			config.Logger = nil

			Expect(config.Validate()).ToNot(Succeed())
		})

		It("requires an engine download URL template", func() {
			config := replayplayer.DefaultAssetManagerConfig
			config.EngineDownloadURLTemplates = nil

			Expect(config.Validate()).ToNot(Succeed())
		})
	})

	It("downloads engines from BAR's CDN first and GitHub second", func() {
		Expect(replayplayer.DefaultAssetManagerConfig.EngineDownloadURLTemplates).To(HaveExactElements(
			HavePrefix("https://files-cdn.beyondallreason.dev/"),
			HavePrefix("https://github.com/beyond-all-reason/RecoilEngine/releases/download/"),
		))
	})
})

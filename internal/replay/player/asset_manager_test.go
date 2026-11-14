package player_test

import (
	"context"
	"path/filepath"
	"runtime"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	replayplayer "github.com/backbone81/bar-replay-analytics-go/internal/replay/player"
	"github.com/backbone81/bar-replay-analytics-go/internal/utils"
)

var _ = Describe("AssetManager", func() {
	Context("NewAssetManager", func() {
		It("rejects an invalid config", func() {
			config := testAssetManagerConfig(newDataDir())
			config.Logger = nil

			Expect(replayplayer.NewAssetManager(config)).Error().To(HaveOccurred())
		})

		It("rejects an engine download URL template that does not parse", func() {
			config := testAssetManagerConfig(newDataDir())
			config.EngineDownloadURLTemplates = []string{"https://example.com/{{.EngineVersion"}

			Expect(replayplayer.NewAssetManager(config)).Error().To(HaveOccurred())
		})

		It("fails when the inventory cannot be read", func() {
			dataDir := newDataDir()
			writeFile(rapidVersionsFile(dataDir), "not gzipped")

			Expect(replayplayer.NewAssetManager(testAssetManagerConfig(dataDir))).Error().To(HaveOccurred())
		})
	})

	Context("assets in the data dir", func() {
		It("finds them without downloading", func(ctx SpecContext) {
			dataDir := newDataDirWithAssets()
			assetManager := newAssetManager(dataDir)

			Expect(assetManager.EnsureEngineVersion(ctx, testEngineVersion)).To(Succeed())
			Expect(assetManager.EnsureMap(ctx, testEngineVersion, testMapName)).To(Succeed())
			Expect(assetManager.EnsureGame(ctx, testEngineVersion, testGameType)).To(Succeed())
			Expect(readPrDownloaderCalls(dataDir)).To(BeEmpty())
		})
	})

	Context("engine download", func() {
		It("downloads and unpacks the engine from the first URL that works", func(ctx SpecContext) {
			dataDir := newDataDir()
			handler, server := newEngineServer()
			assetManager := newAssetManagerWithEngineDownloads(dataDir,
				server.URL+"/missing/{{.EngineVersion}}.7z",
				server.URL+"/engine/recoil_{{.EngineVersion}}_{{.OS}}.7z",
				server.URL+"/engine/never-asked.7z",
			)

			Expect(assetManager.EnsureEngineVersion(ctx, testEngineVersion)).To(Succeed())
			Expect(handler.RequestURIs()).To(HaveExactElements(
				"/missing/"+testEngineVersion+".7z",
				"/engine/recoil_"+testEngineVersion+"_"+runtime.GOOS+".7z",
			))
			Expect(utils.SpringHeadlessFilePath(dataDir, testEngineVersion)).To(BeARegularFile())
			Expect(utils.PrDownloaderFilePath(dataDir, testEngineVersion)).To(BeARegularFile())
			Expect(utils.EngineVersionDir(dataDir, testEngineVersion) + ".7z").NotTo(BeAnExistingFile())
		})

		It("does not download an engine again", func(ctx SpecContext) {
			handler, server := newEngineServer()
			assetManager := newAssetManagerWithEngineDownloads(newDataDir(),
				server.URL+"/engine/{{.EngineVersion}}.7z",
			)
			Expect(assetManager.EnsureEngineVersion(ctx, testEngineVersion)).To(Succeed())

			Expect(assetManager.EnsureEngineVersion(ctx, testEngineVersion)).To(Succeed())
			Expect(handler.RequestURIs()).To(HaveLen(1))
		})

		DescribeTable("rejects an engine version that is not a release tag before downloading anything",
			func(ctx SpecContext, engineVersion string) {
				dataDir := newDataDir()
				handler, server := newEngineServer()
				assetManager := newAssetManagerWithEngineDownloads(dataDir, server.URL+"/engine/{{.EngineVersion}}.7z")

				Expect(assetManager.EnsureEngineVersion(ctx, engineVersion)).ToNot(Succeed())
				Expect(handler.RequestURIs()).To(BeEmpty())
				Expect(utils.EngineDir(dataDir)).NotTo(BeAnExistingFile())
			},
			Entry("a path out of the engine folder", "../../2026.07.04"),
			Entry("URL syntax", "2026.07.04&category=other"),
			Entry("a local build", "local-build"),
			Entry("a tag with extra text", "2026.07.04-rc1"),
			Entry("an empty version", ""),
		)

		It("fails after trying every URL if none works", func(ctx SpecContext) {
			dataDir := newDataDir()
			handler, server := newEngineServer()
			assetManager := newAssetManagerWithEngineDownloads(dataDir,
				server.URL+"/missing/first.7z",
				server.URL+"/missing/second.7z",
			)

			Expect(assetManager.EnsureEngineVersion(ctx, testEngineVersion)).ToNot(Succeed())
			Expect(handler.RequestURIs()).To(HaveExactElements("/missing/first.7z", "/missing/second.7z"))
			Expect(utils.EngineVersionDir(dataDir, testEngineVersion)).NotTo(BeAnExistingFile())
		})

		It("fails if a template cannot be executed", func(ctx SpecContext) {
			assetManager := newAssetManagerWithEngineDownloads(newDataDir(),
				"https://example.com/{{.Missing}}",
			)

			Expect(assetManager.EnsureEngineVersion(ctx, testEngineVersion)).ToNot(Succeed())
		})
	})

	Context("DownloadFile", func() {
		It("saves the response, creating the folder", func(ctx SpecContext) {
			_, server := newEngineServer()
			filePath := filepath.Join(GinkgoT().TempDir(), "new", "engine.7z")

			Expect(newAssetManager(newDataDir()).DownloadFile(ctx, server.URL+"/engine/x.7z", filePath)).
				To(Succeed())

			Expect(readFile(filePath)).To(Equal(readFile(filepath.Join("testdata", "engine.7z"))))
		})

		It("fails on an HTTP error status without writing a file", func(ctx SpecContext) {
			_, server := newEngineServer()
			filePath := filepath.Join(GinkgoT().TempDir(), "engine.7z")

			err := newAssetManager(newDataDir()).DownloadFile(ctx, server.URL+"/missing", filePath)

			Expect(err).To(HaveOccurred())
			Expect(filePath).NotTo(BeAnExistingFile())
		})

		It("removes a partly downloaded file", func(ctx SpecContext) {
			_, server := newEngineServer()
			filePath := filepath.Join(GinkgoT().TempDir(), "engine.7z")

			err := newAssetManager(newDataDir()).DownloadFile(ctx, server.URL+"/truncated/x", filePath)

			Expect(err).To(HaveOccurred())
			Expect(filePath).NotTo(BeAnExistingFile())
		})

		It("stops when the context is canceled", func(ctx SpecContext) {
			_, server := newEngineServer()
			cancelableCtx, cancel := context.WithCancel(ctx)
			cancel()

			err := newAssetManager(newDataDir()).
				DownloadFile(cancelableCtx, server.URL+"/engine/x.7z", filepath.Join(GinkgoT().TempDir(), "engine.7z"))

			Expect(err).To(MatchError(context.Canceled))
		})
	})

	Context("map and game download", func() {
		It("downloads a missing map with the engine's pr-downloader and BAR's servers", func(ctx SpecContext) {
			dataDir := newDataDir()
			addFakeEngine(dataDir, testEngineVersion)
			assetManager := newAssetManager(dataDir)

			Expect(assetManager.EnsureMap(ctx, testEngineVersion, testMapName)).To(Succeed())

			calls := readPrDownloaderCalls(dataDir)
			Expect(calls).To(HaveLen(1))
			Expect(calls[0].Args).To(HaveExactElements(
				"--filesystem-writepath", dataDir,
				"--download-map", testMapName,
			))
			Expect(calls[0].Dir).To(Equal(dataDir))
			Expect(calls[0].Env).To(ConsistOf(
				"PRD_HTTP_SEARCH_URL=https://files-cdn.beyondallreason.dev/find",
				"PRD_RAPID_REPO_MASTER=https://repos-cdn.beyondallreason.dev/repos.gz",
				"PRD_RAPID_USE_STREAMER=false",
			))
		})

		It("downloads a missing game with the engine's pr-downloader", func(ctx SpecContext) {
			dataDir := newDataDir()
			addFakeEngine(dataDir, testEngineVersion)
			assetManager := newAssetManager(dataDir)

			Expect(assetManager.EnsureGame(ctx, testEngineVersion, testGameType)).To(Succeed())

			calls := readPrDownloaderCalls(dataDir)
			Expect(calls).To(HaveLen(1))
			Expect(calls[0].Args).To(HaveExactElements(
				"--filesystem-writepath", dataDir,
				"--download-game", testGameType,
			))
		})

		It("does not download a map or game again", func(ctx SpecContext) {
			dataDir := newDataDir()
			addFakeEngine(dataDir, testEngineVersion)
			assetManager := newAssetManager(dataDir)
			Expect(assetManager.EnsureMap(ctx, testEngineVersion, testMapName)).To(Succeed())
			Expect(assetManager.EnsureGame(ctx, testEngineVersion, testGameType)).To(Succeed())

			Expect(assetManager.EnsureMap(ctx, testEngineVersion, testMapName)).To(Succeed())
			Expect(assetManager.EnsureGame(ctx, testEngineVersion, testGameType)).To(Succeed())

			Expect(readPrDownloaderCalls(dataDir)).To(HaveLen(2))
		})

		It("fails if pr-downloader fails and tries again next time", func(ctx SpecContext) {
			dataDir := newDataDir()
			addFakeEngine(dataDir, testEngineVersion)
			writePrDownloaderBehaviour(dataDir, prDownloaderBehaviour{ExitCode: 3})
			assetManager := newAssetManager(dataDir)

			err := assetManager.EnsureMap(ctx, testEngineVersion, testMapName)

			Expect(err).To(HaveOccurred())
			Expect(assetManager.EnsureMap(ctx, testEngineVersion, testMapName)).ToNot(Succeed())
			Expect(readPrDownloaderCalls(dataDir)).To(HaveLen(2))
		})

		It("rejects an engine version that is not a release tag before running pr-downloader", func(ctx SpecContext) {
			dataDir := newDataDir()
			addFakeEngine(dataDir, testEngineVersion)
			assetManager := newAssetManager(dataDir)

			Expect(assetManager.EnsureMap(ctx, "../recoil_"+testEngineVersion, testMapName)).
				ToNot(Succeed())
			Expect(assetManager.EnsureGame(ctx, "../recoil_"+testEngineVersion, testGameType)).
				ToNot(Succeed())
			Expect(readPrDownloaderCalls(dataDir)).To(BeEmpty())
		})

		It("fails without the engine's pr-downloader", func(ctx SpecContext) {
			assetManager := newAssetManager(newDataDir())

			Expect(assetManager.EnsureMap(ctx, testEngineVersion, testMapName)).
				ToNot(Succeed())
		})

		It("downloads a map that several playbacks need at once only once", func(ctx SpecContext) {
			dataDir := newDataDir()
			addFakeEngine(dataDir, testEngineVersion)
			writePrDownloaderBehaviour(dataDir, prDownloaderBehaviour{Sleep: 200 * time.Millisecond})
			assetManager := newAssetManager(dataDir)

			errs := ensureMapConcurrently(ctx, assetManager, testEngineVersion, testMapName, 8)

			Expect(errs).To(HaveEach(Succeed()))
			Expect(readPrDownloaderCalls(dataDir)).To(HaveLen(1))
		})
	})
})

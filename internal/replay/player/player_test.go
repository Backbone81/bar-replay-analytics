package player_test

import (
	"context"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/backbone81/bar-replay-analytics/internal/events"
	replayplayer "github.com/backbone81/bar-replay-analytics/internal/replay/player"
	"github.com/backbone81/bar-replay-analytics/internal/replay/player/widget"
	"github.com/backbone81/bar-replay-analytics/internal/utils"
)

var _ = Describe("Player", func() {
	Context("New", func() {
		It("rejects an invalid config", func() {
			Expect(replayplayer.New(replayplayer.WithDataDir(newDataDir()))).Error().To(HaveOccurred())
		})
	})

	Context("Prepare", func() {
		It("downloads nothing if the engine, map and game are in the data dir", func(ctx SpecContext) {
			dataDir := newDataDirWithAssets()
			player := newPlayer(dataDir)
			replayInfo, err := utils.ReadReplayInfo(writeTestReplay(demosFile(dataDir, "game.sdfz"), testMapName))
			Expect(err).NotTo(HaveOccurred())

			Expect(player.Prepare(ctx, replayInfo)).To(Succeed())
			Expect(readPrDownloaderCalls(dataDir)).To(BeEmpty())
		})

		It("downloads a missing map, then a missing game", func(ctx SpecContext) {
			dataDir := newDataDir()
			addFakeEngine(dataDir, testEngineVersion)
			player := newPlayer(dataDir)
			replayInfo, err := utils.ReadReplayInfo(writeTestReplay(demosFile(dataDir, "game.sdfz"), testMapName))
			Expect(err).NotTo(HaveOccurred())

			Expect(player.Prepare(ctx, replayInfo)).To(Succeed())
			calls := readPrDownloaderCalls(dataDir)
			Expect(calls).To(HaveLen(2))
			Expect(calls[0].Args).To(ContainElements("--download-map", testMapName))
			Expect(calls[1].Args).To(ContainElements("--download-game", testGameType))
		})

		It("fails if the map cannot be downloaded", func(ctx SpecContext) {
			dataDir := newDataDir()
			addFakeEngine(dataDir, testEngineVersion)
			writePrDownloaderBehaviour(dataDir, prDownloaderBehaviour{ExitCode: 1})
			player := newPlayer(dataDir)
			replayInfo, err := utils.ReadReplayInfo(writeTestReplay(demosFile(dataDir, "game.sdfz"), testMapName))
			Expect(err).NotTo(HaveOccurred())

			Expect(player.Prepare(ctx, replayInfo)).ToNot(Succeed())
			Expect(readPrDownloaderCalls(dataDir)).To(HaveLen(1))
		})
	})

	Context("Run", func() {
		It("runs spring-headless isolated on the replay with a write dir of its own", func(ctx SpecContext) {
			dataDir := newDataDirWithAssets()
			player := newPlayer(dataDir)
			replayInfo := newPreparedReplay(ctx, player, dataDir, "game.sdfz")
			writeSpringHeadlessBehaviour(replayInfo.FilePath, springHeadlessBehaviour{EventsFile: currentEventsFileContent()})

			Expect(player.Run(ctx, replayInfo)).To(Succeed())

			call := readSpringHeadlessCall(replayInfo.FilePath)
			Expect(call.Args).To(HaveLen(6))
			writeDir := call.Args[2]
			Expect(call.Args).To(HaveExactElements(
				"--isolation",
				"--write-dir", writeDir,
				"--config", utils.SpringsettingsCfgFilePath(writeDir),
				replayInfo.FilePath,
			))
			Expect(filepath.Base(writeDir)).To(HavePrefix("bar-replay-analytics-"))
			Expect(writeDir).NotTo(HavePrefix(dataDir))
			Expect(call.Dir).To(Equal(writeDir))
			Expect(call.SpringDataDir).To(Equal(dataDir))
			Expect(writeDir).NotTo(BeAnExistingFile())
		})

		It("passes a relative data dir as an absolute one", func(ctx SpecContext) {
			dataDir := newDataDirWithAssets()
			workingDir, err := os.Getwd()
			Expect(err).NotTo(HaveOccurred())
			relativeDataDir, err := filepath.Rel(workingDir, dataDir)
			Expect(err).NotTo(HaveOccurred())
			player := newPlayer(relativeDataDir)
			replayInfo := newPreparedReplay(ctx, player, dataDir, "game.sdfz")
			writeSpringHeadlessBehaviour(replayInfo.FilePath, springHeadlessBehaviour{EventsFile: currentEventsFileContent()})

			Expect(player.Run(ctx, replayInfo)).To(Succeed())

			Expect(readSpringHeadlessCall(replayInfo.FilePath).SpringDataDir).To(Equal(dataDir))
		})

		It("prepares the write dir with the settings, the archive cache and the analytics widget", func(ctx SpecContext) {
			dataDir := newDataDirWithAssets()
			writeFile(filepath.Join(utils.CacheDir(dataDir), "ArchiveCache20.lua"), "cache")
			writeFile(filepath.Join(utils.CacheDir(dataDir), "other.lua"), "")
			player := newPlayer(dataDir)
			replayInfo := newPreparedReplay(ctx, player, dataDir, "game.sdfz")
			writeSpringHeadlessBehaviour(replayInfo.FilePath, springHeadlessBehaviour{EventsFile: currentEventsFileContent()})

			Expect(player.Run(ctx, replayInfo)).To(Succeed())

			call := readSpringHeadlessCall(replayInfo.FilePath)
			Expect(call.WriteDirFiles).To(ConsistOf(append(
				writeDirWidgetFiles(),
				"springsettings.cfg",
				"cache/ArchiveCache20.lua",
				"LuaUI/Config/BYAR.lua",
			)))
			Expect(call.SpringSettings).To(Equal("HostPortDefault = 0\nWorkerThreadCount = 2"))
		})

		It("enables the analytics widget and the data dir's user widgets in the widget config", func(ctx SpecContext) {
			dataDir := newDataDirWithAssets()
			writeFile(filepath.Join(utils.WidgetsDir(dataDir), "BAR-Advisor", "snd_advisor.lua"),
				"function widget:GetInfo()\n\treturn {\n\t\tname = \"Advisor\",\n\t}\nend\n")
			player := newPlayer(dataDir)
			replayInfo := newPreparedReplay(ctx, player, dataDir, "game.sdfz")
			writeSpringHeadlessBehaviour(replayInfo.FilePath, springHeadlessBehaviour{EventsFile: currentEventsFileContent()})

			Expect(player.Run(ctx, replayInfo)).To(Succeed())

			Expect(readSpringHeadlessCall(replayInfo.FilePath).WidgetConfig).
				To(Equal(string(replayplayer.WidgetConfig([]string{`"Advisor"`, `"` + widget.Name + `"`}))))
		})

		It("leaves the worker thread count to the engine if it is zero", func(ctx SpecContext) {
			dataDir := newDataDirWithAssets()
			player := newPlayer(dataDir, replayplayer.WithWorkerThreadCount(0))
			replayInfo := newPreparedReplay(ctx, player, dataDir, "game.sdfz")
			writeSpringHeadlessBehaviour(replayInfo.FilePath, springHeadlessBehaviour{EventsFile: currentEventsFileContent()})

			Expect(player.Run(ctx, replayInfo)).To(Succeed())

			Expect(readSpringHeadlessCall(replayInfo.FilePath).SpringSettings).To(Equal("HostPortDefault = 0"))
		})

		It("copies the events file next to the replay and writes the engine's output to the log", func(ctx SpecContext) {
			dataDir := newDataDirWithAssets()
			player := newPlayer(dataDir)
			replayInfo := newPreparedReplay(ctx, player, dataDir, "game.sdfz")
			writeFile(utils.DemoLogFilePath(replayInfo.FilePath), "an older and longer log of an earlier playback\n")
			writeSpringHeadlessBehaviour(replayInfo.FilePath, springHeadlessBehaviour{
				Output:     "[f=0000000] engine output\n",
				EventsFile: currentEventsFileContent(),
			})

			Expect(player.Run(ctx, replayInfo)).To(Succeed())

			Expect(readFile(utils.DemoEventsFilePath(replayInfo.FilePath))).To(Equal(currentEventsFileContent()))
			Expect(readFile(utils.DemoLogFilePath(replayInfo.FilePath))).To(Equal("[f=0000000] engine output\n"))
		})

		It("fails if the engine fails and keeps its output in the log", func(ctx SpecContext) {
			dataDir := newDataDirWithAssets()
			player := newPlayer(dataDir)
			replayInfo := newPreparedReplay(ctx, player, dataDir, "game.sdfz")
			writeSpringHeadlessBehaviour(replayInfo.FilePath, springHeadlessBehaviour{
				ExitCode: 1,
				Output:   "Error: map not found\n",
			})

			err := player.Run(ctx, replayInfo)

			Expect(err).To(HaveOccurred())
			Expect(readFile(utils.DemoLogFilePath(replayInfo.FilePath))).To(Equal("Error: map not found\n"))
			Expect(utils.DemoEventsFilePath(replayInfo.FilePath)).NotTo(BeAnExistingFile())
		})

		It("fails if the analytics widget wrote no events file", func(ctx SpecContext) {
			dataDir := newDataDirWithAssets()
			player := newPlayer(dataDir)
			replayInfo := newPreparedReplay(ctx, player, dataDir, "game.sdfz")

			Expect(player.Run(ctx, replayInfo)).To(MatchError(os.ErrNotExist))
			Expect(utils.DemoEventsFilePath(replayInfo.FilePath)).NotTo(BeAnExistingFile())
		})

		DescribeTable("fails and does not copy an events file that is not complete and current",
			func(ctx SpecContext, eventsFile string) {
				dataDir := newDataDirWithAssets()
				player := newPlayer(dataDir)
				replayInfo := newPreparedReplay(ctx, player, dataDir, "game.sdfz")
				writeSpringHeadlessBehaviour(replayInfo.FilePath, springHeadlessBehaviour{EventsFile: eventsFile})

				Expect(player.Run(ctx, replayInfo)).ToNot(Succeed())
				Expect(utils.DemoEventsFilePath(replayInfo.FilePath)).NotTo(BeAnExistingFile())
			},
			Entry("without an end", eventsFileContent(widget.FormatVersion, "")),
			Entry("of a game that did not end", eventsFileContent(widget.FormatVersion, events.EndReasonStalled)),
			Entry("cut off", currentEventsFileContent()[:10]),
			Entry("of an older format version", eventsFileContent(widget.FormatVersion-1, events.EndReasonGameOver)),
		)

		It("fails without the engine", func(ctx SpecContext) {
			dataDir := newDataDirWithAssets()
			player := newPlayer(dataDir)
			replayInfo := newPreparedReplay(ctx, player, dataDir, "game.sdfz")
			Expect(os.RemoveAll(utils.EngineVersionDir(dataDir, testEngineVersion))).To(Succeed())

			Expect(player.Run(ctx, replayInfo)).ToNot(Succeed())
		})

		It("stops the engine when the context is canceled", func(ctx SpecContext) {
			dataDir := newDataDirWithAssets()
			player := newPlayer(dataDir)
			replayInfo := newPreparedReplay(ctx, player, dataDir, "game.sdfz")
			writeSpringHeadlessBehaviour(replayInfo.FilePath, springHeadlessBehaviour{WaitUntilStopped: true})
			cancelableCtx, cancel := context.WithCancel(ctx)
			defer cancel()

			result := runPlayerInBackground(cancelableCtx, player, replayInfo)
			Eventually(replaySibling).WithArguments(replayInfo.FilePath, springHeadlessCallSuffix).Should(BeAnExistingFile())
			cancel()

			Eventually(result).Should(Receive(MatchError(context.Canceled)))
		})
	})
})

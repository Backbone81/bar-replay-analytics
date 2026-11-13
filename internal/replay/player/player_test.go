package player_test

import (
	"context"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/backbone81/bar-replay-analytics-go/internal/events"
	replayplayer "github.com/backbone81/bar-replay-analytics-go/internal/replay/player"
	"github.com/backbone81/bar-replay-analytics-go/internal/replay/player/widget"
	"github.com/backbone81/bar-replay-analytics-go/internal/utils"
)

var _ = Describe("Player", func() {
	Context("New", func() {
		It("rejects an invalid config", func() {
			Expect(replayplayer.New(replayplayer.WithDataDir(newDataDir()))).Error().To(HaveOccurred())
		})
	})

	Context("Prepare", func() {
		It("returns a copy of the replay info with the engine folder", func(ctx SpecContext) {
			dataDir := newDataDirWithAssets()
			player := newPlayer(dataDir)
			replayInfo, err := replayplayer.ReadReplayInfo(writeTestReplay(demosFile(dataDir, "game.sdfz"), testMapName))
			Expect(err).NotTo(HaveOccurred())

			preparedReplayInfo, err := player.Prepare(ctx, replayInfo)

			Expect(err).NotTo(HaveOccurred())
			Expect(preparedReplayInfo.EngineDir).To(Equal(engineDir(dataDir, testEngineVersion)))
			Expect(replayInfo.EngineDir).To(BeEmpty())
			Expect(preparedReplayInfo.FilePath).To(Equal(replayInfo.FilePath))
		})

		It("downloads a missing map, then a missing game", func(ctx SpecContext) {
			dataDir := newDataDir()
			addFakeEngine(dataDir, testEngineVersion)
			player := newPlayer(dataDir)
			replayInfo, err := replayplayer.ReadReplayInfo(writeTestReplay(demosFile(dataDir, "game.sdfz"), testMapName))
			Expect(err).NotTo(HaveOccurred())

			Expect(player.Prepare(ctx, replayInfo)).Error().NotTo(HaveOccurred())
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
			replayInfo, err := replayplayer.ReadReplayInfo(writeTestReplay(demosFile(dataDir, "game.sdfz"), testMapName))
			Expect(err).NotTo(HaveOccurred())

			Expect(player.Prepare(ctx, replayInfo)).Error().To(HaveOccurred())
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
				"--config", filepath.Join(writeDir, "springsettings.cfg"),
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
			writeFile(filepath.Join(dataDir, utils.CacheDir, "ArchiveCache20.lua"), "cache")
			writeFile(filepath.Join(dataDir, utils.CacheDir, "other.lua"), "")
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
			writeFile(filepath.Join(dataDir, utils.WidgetsDir, "BAR-Advisor", "snd_advisor.lua"),
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
			writeFile(replaySibling(replayInfo.FilePath, ".log"), "an older and longer log of an earlier playback\n")
			writeSpringHeadlessBehaviour(replayInfo.FilePath, springHeadlessBehaviour{
				Output:     "[f=0000000] engine output\n",
				EventsFile: currentEventsFileContent(),
			})

			Expect(player.Run(ctx, replayInfo)).To(Succeed())

			Expect(readFile(replaySibling(replayInfo.FilePath, "_events.json"))).To(Equal(currentEventsFileContent()))
			Expect(readFile(replaySibling(replayInfo.FilePath, ".log"))).To(Equal("[f=0000000] engine output\n"))
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
			Expect(readFile(replaySibling(replayInfo.FilePath, ".log"))).To(Equal("Error: map not found\n"))
			Expect(replaySibling(replayInfo.FilePath, "_events.json")).NotTo(BeAnExistingFile())
		})

		It("fails if the analytics widget wrote no events file", func(ctx SpecContext) {
			dataDir := newDataDirWithAssets()
			player := newPlayer(dataDir)
			replayInfo := newPreparedReplay(ctx, player, dataDir, "game.sdfz")

			Expect(player.Run(ctx, replayInfo)).To(MatchError(os.ErrNotExist))
			Expect(replaySibling(replayInfo.FilePath, "_events.json")).NotTo(BeAnExistingFile())
		})

		DescribeTable("fails and does not copy an events file that is not complete and current",
			func(ctx SpecContext, eventsFile string) {
				dataDir := newDataDirWithAssets()
				player := newPlayer(dataDir)
				replayInfo := newPreparedReplay(ctx, player, dataDir, "game.sdfz")
				writeSpringHeadlessBehaviour(replayInfo.FilePath, springHeadlessBehaviour{EventsFile: eventsFile})

				Expect(player.Run(ctx, replayInfo)).ToNot(Succeed())
				Expect(replaySibling(replayInfo.FilePath, "_events.json")).NotTo(BeAnExistingFile())
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
			replayInfo.EngineDir = filepath.Join(dataDir, "engine", "missing")

			Expect(player.Run(ctx, replayInfo)).
				ToNot(Succeed())
		})

		It("asks the engine to quit when the context is canceled", func(ctx SpecContext) {
			dataDir := newDataDirWithAssets()
			player := newPlayer(dataDir)
			replayInfo := newPreparedReplay(ctx, player, dataDir, "game.sdfz")
			writeSpringHeadlessBehaviour(replayInfo.FilePath, springHeadlessBehaviour{WaitForSignal: true})
			cancelableCtx, cancel := context.WithCancel(ctx)
			defer cancel()

			result := runPlayerInBackground(cancelableCtx, player, replayInfo)
			Eventually(replaySibling).WithArguments(replayInfo.FilePath, springHeadlessCallSuffix).Should(BeAnExistingFile())
			cancel()

			Eventually(result).Should(Receive(MatchError(context.Canceled)))
			Expect(readFile(replaySibling(replayInfo.FilePath, springHeadlessSignalSuffix))).To(Equal("terminated"))
		})
	})

	Context("ValidateEventsFile", func() {
		It("fails without an events file", func() {
			eventsFilePath := filepath.Join(GinkgoT().TempDir(), "game_events.json")

			Expect(replayplayer.ValidateEventsFile(eventsFilePath)).To(MatchError(os.ErrNotExist))
		})

		DescribeTable("accepts a complete events file of at least the widget's format version",
			func(formatVersion int) {
				eventsFilePath := filepath.Join(GinkgoT().TempDir(), "game_events.json")
				writeFile(eventsFilePath, eventsFileContent(formatVersion, events.EndReasonGameOver))

				Expect(replayplayer.ValidateEventsFile(eventsFilePath)).To(Succeed())
			},
			Entry("the same version", widget.FormatVersion),
			Entry("a newer version", widget.FormatVersion+1),
		)

		DescribeTable("rejects an events file that is not complete and current",
			func(eventsFile string) {
				eventsFilePath := filepath.Join(GinkgoT().TempDir(), "game_events.json")
				writeFile(eventsFilePath, eventsFile)

				Expect(replayplayer.ValidateEventsFile(eventsFilePath)).ToNot(Succeed())
			},
			Entry("without an end", eventsFileContent(widget.FormatVersion, "")),
			Entry("of a game that did not end", eventsFileContent(widget.FormatVersion, events.EndReasonStalled)),
			Entry("cut off", currentEventsFileContent()[:10]),
			Entry("of an older format version", eventsFileContent(widget.FormatVersion-1, events.EndReasonGameOver)),
		)
	})
})

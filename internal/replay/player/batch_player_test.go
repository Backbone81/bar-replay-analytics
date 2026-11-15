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

var _ = Describe("BatchPlayer", func() {
	Context("NewBatchPlayer", func() {
		It("replaces zero concurrent replays by a number of its own", func() {
			dataDir := newDataDir()

			Expect(replayplayer.NewBatchPlayer(
				replayplayer.WithDataDir(dataDir),
				replayplayer.WithAssetManager(newAssetManager(dataDir)),
				replayplayer.WithConcurrentReplays(0),
			)).Error().NotTo(HaveOccurred())
		})
	})

	Context("IsEventsFileCurrent", func() {
		It("is false without an events file", func() {
			replayFilePath := filepath.Join(GinkgoT().TempDir(), "game.sdfz")

			Expect(newBatchPlayer(newDataDir()).IsEventsFileCurrent(replayFilePath)).To(BeFalse())
		})

		DescribeTable("compares a complete events file with the widget's format version",
			func(formatVersion int, expected bool) {
				replayFilePath := filepath.Join(GinkgoT().TempDir(), "game.sdfz")
				writeFile(utils.DemoEventsFilePath(replayFilePath),
					eventsFileContent(formatVersion, events.EndReasonGameOver))

				Expect(newBatchPlayer(newDataDir()).IsEventsFileCurrent(replayFilePath)).To(Equal(expected))
			},
			Entry("an older version is outdated", widget.FormatVersion-1, false),
			Entry("the same version is current", widget.FormatVersion, true),
			Entry("a newer version is current", widget.FormatVersion+1, true),
		)

		It("is false for an events file without an end", func() {
			replayFilePath := filepath.Join(GinkgoT().TempDir(), "game.sdfz")
			writeFile(utils.DemoEventsFilePath(replayFilePath), eventsFileContent(widget.FormatVersion, ""))

			Expect(newBatchPlayer(newDataDir()).IsEventsFileCurrent(replayFilePath)).To(BeFalse())
		})

		It("is false for an events file of a game that did not end", func() {
			replayFilePath := filepath.Join(GinkgoT().TempDir(), "game.sdfz")
			writeFile(utils.DemoEventsFilePath(replayFilePath),
				eventsFileContent(widget.FormatVersion, events.EndReasonStalled))

			Expect(newBatchPlayer(newDataDir()).IsEventsFileCurrent(replayFilePath)).To(BeFalse())
		})

		It("is false for an events file that was cut off", func() {
			replayFilePath := filepath.Join(GinkgoT().TempDir(), "game.sdfz")
			writeFile(utils.DemoEventsFilePath(replayFilePath), currentEventsFileContent()[:10])

			Expect(newBatchPlayer(newDataDir()).IsEventsFileCurrent(replayFilePath)).To(BeFalse())
		})
	})

	Context("Run", func() {
		It("plays every replay back and writes their events files", func(ctx SpecContext) {
			dataDir := newDataDirWithAssets()
			first := writePlayableReplay(dataDir, "a.sdfz", testMapName)
			second := writePlayableReplay(dataDir, "b.sdfz", testMapName)

			Expect(newBatchPlayer(dataDir).Run(ctx, ".")).To(Succeed())

			Expect(readFile(utils.DemoEventsFilePath(first))).To(Equal(currentEventsFileContent()))
			Expect(readFile(utils.DemoEventsFilePath(second))).To(Equal(currentEventsFileContent()))
		})

		It("skips a replay whose events file is current and plays one whose events file is outdated", func(ctx SpecContext) {
			dataDir := newDataDirWithAssets()
			current := writePlayableReplay(dataDir, "current.sdfz", testMapName)
			writeFile(utils.DemoEventsFilePath(current), currentEventsFileContent())
			outdated := writePlayableReplay(dataDir, "outdated.sdfz", testMapName)
			writeFile(utils.DemoEventsFilePath(outdated),
				eventsFileContent(widget.FormatVersion-1, events.EndReasonGameOver))

			Expect(newBatchPlayer(dataDir).Run(ctx, ".")).To(Succeed())

			Expect(springHeadlessCalled(current)).To(BeFalse())
			Expect(springHeadlessCalled(outdated)).To(BeTrue())
			Expect(readFile(utils.DemoEventsFilePath(outdated))).To(Equal(currentEventsFileContent()))
		})

		It("plays only the replays the filter matches", func(ctx SpecContext) {
			dataDir := newDataDirWithAssets()
			matching := writePlayableReplay(dataDir, "a.sdfz", testMapName)
			other := writePlayableReplay(dataDir, "b.sdfz", "Supreme Isthmus v1.8")

			Expect(newBatchPlayer(dataDir, replayplayer.WithFilter(utils.Filter{MapNames: []string{"strata"}})).
				Run(ctx, ".")).To(Succeed())

			Expect(springHeadlessCalled(matching)).To(BeTrue())
			Expect(springHeadlessCalled(other)).To(BeFalse())
		})

		It("plays at most the maximum number of replays, the first in file path order", func(ctx SpecContext) {
			dataDir := newDataDirWithAssets()
			first := writePlayableReplay(dataDir, "a.sdfz", testMapName)
			second := writePlayableReplay(dataDir, "b.sdfz", testMapName)
			third := writePlayableReplay(dataDir, "c.sdfz", testMapName)

			Expect(newBatchPlayer(dataDir, replayplayer.WithMaxReplays(2)).Run(ctx, ".")).
				To(Succeed())

			Expect(springHeadlessCalled(first)).To(BeTrue())
			Expect(springHeadlessCalled(second)).To(BeTrue())
			Expect(springHeadlessCalled(third)).To(BeFalse())
		})

		It("counts a failed replay towards the maximum number of replays", func(ctx SpecContext) {
			dataDir := newDataDirWithAssets()
			failing := writeTestReplay(demosFile(dataDir, "a.sdfz"), testMapName)
			writeSpringHeadlessBehaviour(failing, springHeadlessBehaviour{ExitCode: 1})
			second := writePlayableReplay(dataDir, "b.sdfz", testMapName)
			third := writePlayableReplay(dataDir, "c.sdfz", testMapName)

			Expect(newBatchPlayer(dataDir, replayplayer.WithMaxReplays(2)).Run(ctx, ".")).
				ToNot(Succeed())

			Expect(springHeadlessCalled(second)).To(BeTrue())
			Expect(springHeadlessCalled(third)).To(BeFalse())
		})

		It("plays the other replays when some fail and counts the failures", func(ctx SpecContext) {
			dataDir := newDataDirWithAssets()
			writeFile(demosFile(dataDir, "a.sdfz"), "no replay")
			failing := writeTestReplay(demosFile(dataDir, "b.sdfz"), testMapName)
			writeSpringHeadlessBehaviour(failing, springHeadlessBehaviour{ExitCode: 1})
			unpreparable := demosFile(dataDir, "c.sdfz")
			writeReplay(unpreparable, "2026.01.01", testSetupScript(testMapName, testGameType), 16, testGameTime)
			working := writePlayableReplay(dataDir, "d.sdfz", testMapName)

			Expect(newBatchPlayer(dataDir).Run(ctx, ".")).
				ToNot(Succeed())

			Expect(readFile(utils.DemoEventsFilePath(working))).To(Equal(currentEventsFileContent()))
		})

		It("plays the replays in parallel", func(ctx SpecContext) {
			dataDir := newDataDirWithAssets()
			first := writeTestReplay(demosFile(dataDir, "a.sdfz"), testMapName)
			second := writeTestReplay(demosFile(dataDir, "b.sdfz"), testMapName)
			// Each fake engine waits until the other one started, so the playbacks only finish if they run at once.
			writeSpringHeadlessBehaviour(first, springHeadlessBehaviour{
				EventsFile:  currentEventsFileContent(),
				WaitForFile: replaySibling(second, springHeadlessCallSuffix),
			})
			writeSpringHeadlessBehaviour(second, springHeadlessBehaviour{
				EventsFile:  currentEventsFileContent(),
				WaitForFile: replaySibling(first, springHeadlessCallSuffix),
			})

			Expect(newBatchPlayer(dataDir, replayplayer.WithConcurrentReplays(2)).Run(ctx, ".")).
				To(Succeed())
		})

		It("fails for a missing path without playing anything", func(ctx SpecContext) {
			dataDir := newDataDirWithAssets()
			replayFilePath := writePlayableReplay(dataDir, "a.sdfz", testMapName)

			Expect(newBatchPlayer(dataDir).Run(ctx, "a.sdfz", "missing.sdfz")).
				To(MatchError(os.ErrNotExist))

			Expect(springHeadlessCalled(replayFilePath)).To(BeFalse())
		})

		It("plays nothing with a canceled context", func(ctx SpecContext) {
			dataDir := newDataDirWithAssets()
			replayFilePath := writePlayableReplay(dataDir, "a.sdfz", testMapName)
			cancelableCtx, cancel := context.WithCancel(ctx)
			cancel()

			Expect(newBatchPlayer(dataDir).Run(cancelableCtx, ".")).To(MatchError(context.Canceled))

			Expect(springHeadlessCalled(replayFilePath)).To(BeFalse())
		})

		It("stops the playbacks and plays no more replays when the context is canceled", func(ctx SpecContext) {
			dataDir := newDataDirWithAssets()
			waiting := writeTestReplay(demosFile(dataDir, "a.sdfz"), testMapName)
			writeSpringHeadlessBehaviour(waiting, springHeadlessBehaviour{WaitForSignal: true})
			next := writePlayableReplay(dataDir, "b.sdfz", testMapName)
			cancelableCtx, cancel := context.WithCancel(ctx)
			defer cancel()

			result := runBatchPlayerInBackground(cancelableCtx, newBatchPlayer(dataDir), ".")
			Eventually(replaySibling).WithArguments(waiting, springHeadlessCallSuffix).Should(BeAnExistingFile())
			cancel()

			Eventually(result).Should(Receive(MatchError(context.Canceled)))
			Expect(replaySibling(waiting, springHeadlessSignalSuffix)).To(BeAnExistingFile())
			Expect(springHeadlessCalled(next)).To(BeFalse())
		})
	})
})

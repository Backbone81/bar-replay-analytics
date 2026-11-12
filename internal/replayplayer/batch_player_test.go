package replayplayer_test

import (
	"context"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/backbone81/bar-replay-analytics-go/internal/events"
	"github.com/backbone81/bar-replay-analytics-go/internal/replayplayer"
	"github.com/backbone81/bar-replay-analytics-go/internal/replayplayer/widget"
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

		It("rejects an invalid config", func() {
			Expect(replayplayer.NewBatchPlayer(replayplayer.WithDataDir(newDataDir()))).Error().To(HaveOccurred())
		})
	})

	Context("ResolvePaths", func() {
		It("resolves a relative path in the demos folder of the data dir", func() {
			dataDir := newDataDir()
			writeFile(demosFile(dataDir, "game.sdfz"), "")

			Expect(newBatchPlayer(dataDir).ResolvePaths([]string{"game.sdfz"})).
				To(HaveExactElements(demosFile(dataDir, "game.sdfz")))
		})

		It("uses an absolute path as it is", func() {
			filePath := filepath.Join(GinkgoT().TempDir(), "game.sdfz")
			writeFile(filePath, "")

			Expect(newBatchPlayer(newDataDir()).ResolvePaths([]string{filePath})).To(HaveExactElements(filePath))
		})

		It("resolves a folder to the replay files directly in it", func() {
			dataDir := newDataDir()
			writeFile(demosFile(dataDir, "b.sdfz"), "")
			writeFile(demosFile(dataDir, "a.sdfz"), "")
			writeFile(demosFile(dataDir, "a.log"), "")
			writeFile(demosFile(dataDir, filepath.Join("leaderboard", "c.sdfz")), "")
			Expect(os.Mkdir(demosFile(dataDir, "folder.sdfz"), 0o750)).To(Succeed())

			Expect(newBatchPlayer(dataDir).ResolvePaths([]string{"."})).To(HaveExactElements(
				demosFile(dataDir, "a.sdfz"),
				demosFile(dataDir, "b.sdfz"),
			))
		})

		It("sorts the replays and returns each once", func() {
			dataDir := newDataDir()
			writeFile(demosFile(dataDir, "b.sdfz"), "")
			writeFile(demosFile(dataDir, "a.sdfz"), "")

			Expect(newBatchPlayer(dataDir).ResolvePaths([]string{"b.sdfz", ".", demosFile(dataDir, "a.sdfz")})).
				To(HaveExactElements(
					demosFile(dataDir, "a.sdfz"),
					demosFile(dataDir, "b.sdfz"),
				))
		})

		It("fails for a missing path", func() {
			Expect(newBatchPlayer(newDataDir()).ResolvePaths([]string{"missing.sdfz"})).Error().To(MatchError(os.ErrNotExist))
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
				writeFile(replaySibling(replayFilePath, "_events.json"),
					eventsFileContent(formatVersion, events.EndReasonGameOver))

				Expect(newBatchPlayer(newDataDir()).IsEventsFileCurrent(replayFilePath)).To(Equal(expected))
			},
			Entry("an older version is outdated", widget.FormatVersion-1, false),
			Entry("the same version is current", widget.FormatVersion, true),
			Entry("a newer version is current", widget.FormatVersion+1, true),
		)

		It("is false for an events file without an end", func() {
			replayFilePath := filepath.Join(GinkgoT().TempDir(), "game.sdfz")
			writeFile(replaySibling(replayFilePath, "_events.json"), eventsFileContent(widget.FormatVersion, ""))

			Expect(newBatchPlayer(newDataDir()).IsEventsFileCurrent(replayFilePath)).To(BeFalse())
		})

		It("is false for an events file of a game that did not end", func() {
			replayFilePath := filepath.Join(GinkgoT().TempDir(), "game.sdfz")
			writeFile(replaySibling(replayFilePath, "_events.json"),
				eventsFileContent(widget.FormatVersion, events.EndReasonStalled))

			Expect(newBatchPlayer(newDataDir()).IsEventsFileCurrent(replayFilePath)).To(BeFalse())
		})

		It("is false for an events file that was cut off", func() {
			replayFilePath := filepath.Join(GinkgoT().TempDir(), "game.sdfz")
			writeFile(replaySibling(replayFilePath, "_events.json"), currentEventsFileContent()[:10])

			Expect(newBatchPlayer(newDataDir()).IsEventsFileCurrent(replayFilePath)).To(BeFalse())
		})
	})

	Context("Run", func() {
		It("plays every replay back and writes their events files", func(ctx SpecContext) {
			dataDir := newDataDirWithAssets()
			first := writePlayableReplay(dataDir, "a.sdfz", testMapName)
			second := writePlayableReplay(dataDir, "b.sdfz", testMapName)

			Expect(newBatchPlayer(dataDir).Run(ctx, ".")).To(Succeed())

			Expect(readFile(replaySibling(first, "_events.json"))).To(Equal(currentEventsFileContent()))
			Expect(readFile(replaySibling(second, "_events.json"))).To(Equal(currentEventsFileContent()))
		})

		It("skips a replay whose events file is current and plays one whose events file is outdated", func(ctx SpecContext) {
			dataDir := newDataDirWithAssets()
			current := writePlayableReplay(dataDir, "current.sdfz", testMapName)
			writeFile(replaySibling(current, "_events.json"), currentEventsFileContent())
			outdated := writePlayableReplay(dataDir, "outdated.sdfz", testMapName)
			writeFile(replaySibling(outdated, "_events.json"),
				eventsFileContent(widget.FormatVersion-1, events.EndReasonGameOver))

			Expect(newBatchPlayer(dataDir).Run(ctx, ".")).To(Succeed())

			Expect(springHeadlessCalled(current)).To(BeFalse())
			Expect(springHeadlessCalled(outdated)).To(BeTrue())
			Expect(readFile(replaySibling(outdated, "_events.json"))).To(Equal(currentEventsFileContent()))
		})

		It("plays only the replays the filter matches", func(ctx SpecContext) {
			dataDir := newDataDirWithAssets()
			matching := writePlayableReplay(dataDir, "a.sdfz", testMapName)
			other := writePlayableReplay(dataDir, "b.sdfz", "Supreme Isthmus v1.8")

			Expect(newBatchPlayer(dataDir, replayplayer.WithFilter(replayplayer.Filter{MapNames: []string{"strata"}})).
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
			writeReplay(unpreparable, "2026.01.01", testSetupScript(testMapName, testGameType), 16)
			working := writePlayableReplay(dataDir, "d.sdfz", testMapName)

			Expect(newBatchPlayer(dataDir).Run(ctx, ".")).
				ToNot(Succeed())

			Expect(readFile(replaySibling(working, "_events.json"))).To(Equal(currentEventsFileContent()))
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

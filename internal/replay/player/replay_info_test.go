package player_test

import (
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	replayplayer "github.com/backbone81/bar-replay-analytics-go/internal/replay/player"
	"github.com/backbone81/bar-replay-analytics-go/internal/tdf"
)

var _ = Describe("ReplayInfo", func() {
	Context("ReadReplayInfo", func() {
		It("reads the header and the start script", func() {
			filePath := writeTestReplay(filepath.Join(GinkgoT().TempDir(), "game.sdfz"), testMapName)

			replayInfo, err := replayplayer.ReadReplayInfo(filePath)

			Expect(err).NotTo(HaveOccurred())
			Expect(replayInfo).To(Equal(&replayplayer.ReplayInfo{
				FilePath:      filePath,
				EngineVersion: testEngineVersion,
				MapName:       testMapName,
				GameType:      testGameType,
				PlayerNames:   []string{"Alice", "BARb Hard", "SimpleAI"},
				Duration:      testGameTime * time.Second,
			}))
		})

		It("makes the file path absolute", func() {
			filePath := writeTestReplay(filepath.Join(GinkgoT().TempDir(), "game.sdfz"), testMapName)
			workingDir, err := os.Getwd()
			Expect(err).NotTo(HaveOccurred())
			relativePath, err := filepath.Rel(workingDir, filePath)
			Expect(err).NotTo(HaveOccurred())

			replayInfo, err := replayplayer.ReadReplayInfo(relativePath)

			Expect(err).NotTo(HaveOccurred())
			Expect(replayInfo.FilePath).To(Equal(filePath))
		})

		It("rejects a replay without a recorded stream", func() {
			filePath := filepath.Join(GinkgoT().TempDir(), "crashed.sdfz")
			writeReplay(filePath, testEngineVersion, testSetupScript(testMapName, testGameType), 0, testGameTime)

			Expect(replayplayer.ReadReplayInfo(filePath)).Error().To(HaveOccurred())
		})

		It("rejects a replay without a game time", func() {
			filePath := filepath.Join(GinkgoT().TempDir(), "not-ended.sdfz")
			writeReplay(filePath, testEngineVersion, testSetupScript(testMapName, testGameType), 16, 0)

			Expect(replayplayer.ReadReplayInfo(filePath)).Error().To(HaveOccurred())
		})

		It("rejects a start script without gametype", func() {
			filePath := filepath.Join(GinkgoT().TempDir(), "game.sdfz")
			writeReplay(filePath, testEngineVersion, "[game]\n{\n\tmapname="+testMapName+";\n}\n", 16, testGameTime)

			Expect(replayplayer.ReadReplayInfo(filePath)).Error().To(HaveOccurred())
		})

		It("rejects a start script without mapname", func() {
			filePath := filepath.Join(GinkgoT().TempDir(), "game.sdfz")
			writeReplay(filePath, testEngineVersion, "[game]\n{\n\tgametype="+testGameType+";\n}\n", 16, testGameTime)

			Expect(replayplayer.ReadReplayInfo(filePath)).Error().To(HaveOccurred())
		})

		It("rejects a file that is no replay", func() {
			filePath := filepath.Join(GinkgoT().TempDir(), "game.sdfz")
			writeFile(filePath, "no replay")

			Expect(replayplayer.ReadReplayInfo(filePath)).Error().To(HaveOccurred())
		})

		It("fails for a missing file", func() {
			Expect(replayplayer.ReadReplayInfo(filepath.Join(GinkgoT().TempDir(), "missing.sdfz"))).Error().
				To(MatchError(os.ErrNotExist))
		})
	})

	Context("GetPlayerNames", func() {
		It("returns the players in order, then the AIs, without the spectators", func() {
			setupScript, err := tdf.Parse(testSetupScript(testMapName, testGameType))
			Expect(err).NotTo(HaveOccurred())

			Expect(replayplayer.GetPlayerNames(setupScript.Section("game"))).
				To(HaveExactElements("Alice", "BARb Hard", "SimpleAI"))
		})

		It("counts a player without a spectator value as playing", func() {
			setupScript, err := tdf.Parse("[game]\n{\n\t[player0]\n\t{\n\t\tname=Alice;\n\t}\n}\n")
			Expect(err).NotTo(HaveOccurred())

			Expect(replayplayer.GetPlayerNames(setupScript.Section("game"))).To(HaveExactElements("Alice"))
		})

		It("returns no names for a game without players and AIs", func() {
			setupScript, err := tdf.Parse("[game]\n{\n\tmapname=" + testMapName + ";\n}\n")
			Expect(err).NotTo(HaveOccurred())

			Expect(replayplayer.GetPlayerNames(setupScript.Section("game"))).To(BeEmpty())
		})

		It("rejects a spectator value that is no boolean", func() {
			setupScript, err := tdf.Parse("[game]\n{\n\t[player3]\n\t{\n\t\tname=Alice;\n\t\tspectator=maybe;\n\t}\n}\n")
			Expect(err).NotTo(HaveOccurred())

			Expect(replayplayer.GetPlayerNames(setupScript.Section("game"))).Error().To(HaveOccurred())
		})
	})
})

package utils_test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/backbone81/bar-replay-analytics/internal/utils"
)

var _ = Describe("Replays", func() {
	Context("ResolveReplayPaths", func() {
		It("resolves a relative path in the demos folder of the data dir", func() {
			dataDir := newDataDir()
			writeFile(demosFile(dataDir, "game.sdfz"), "")

			Expect(utils.ResolveReplayPaths(dataDir, []string{"game.sdfz"})).
				To(HaveExactElements(demosFile(dataDir, "game.sdfz")))
		})

		It("uses an absolute path as it is", func() {
			filePath := filepath.Join(GinkgoT().TempDir(), "game.sdfz")
			writeFile(filePath, "")

			Expect(utils.ResolveReplayPaths(newDataDir(), []string{filePath})).To(HaveExactElements(filePath))
		})

		It("resolves a folder to the replay files directly in it", func() {
			dataDir := newDataDir()
			writeFile(demosFile(dataDir, "b.sdfz"), "")
			writeFile(demosFile(dataDir, "a.sdfz"), "")
			writeFile(demosFile(dataDir, "a.log"), "")
			writeFile(demosFile(dataDir, filepath.Join("leaderboard", "c.sdfz")), "")
			Expect(os.Mkdir(demosFile(dataDir, "folder.sdfz"), 0o750)).To(Succeed())

			Expect(utils.ResolveReplayPaths(dataDir, []string{"."})).To(HaveExactElements(
				demosFile(dataDir, "a.sdfz"),
				demosFile(dataDir, "b.sdfz"),
			))
		})

		It("sorts the replays and returns each once", func() {
			dataDir := newDataDir()
			writeFile(demosFile(dataDir, "b.sdfz"), "")
			writeFile(demosFile(dataDir, "a.sdfz"), "")

			Expect(utils.ResolveReplayPaths(dataDir, []string{"b.sdfz", ".", demosFile(dataDir, "a.sdfz")})).
				To(HaveExactElements(
					demosFile(dataDir, "a.sdfz"),
					demosFile(dataDir, "b.sdfz"),
				))
		})

		It("fails for a missing path", func() {
			Expect(utils.ResolveReplayPaths(newDataDir(), []string{"missing.sdfz"})).Error().To(MatchError(os.ErrNotExist))
		})
	})
})

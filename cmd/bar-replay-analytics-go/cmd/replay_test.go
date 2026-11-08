package cmd_test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/backbone81/bar-replay-analytics-go/cmd/bar-replay-analytics-go/cmd"
)

var _ = Describe("replayPaths", func() {
	var demosDir string

	BeforeEach(func() {
		dataDir := GinkgoT().TempDir()
		demosDir = filepath.Join(dataDir, "demos")
		DeferCleanup(cmd.SetBarDataDir(dataDir))
		for _, filePath := range []string{
			"a.sdfz", "a_events.json", "leaderboard/c.sdfz", "leaderboard/b.sdfz", "leaderboard/b_report.json",
			"leaderboard/deeper/d.sdfz",
		} {
			filePath = filepath.Join(demosDir, filePath)
			Expect(os.MkdirAll(filepath.Dir(filePath), 0o750)).To(Succeed())
			Expect(os.WriteFile(filePath, nil, 0o600)).To(Succeed())
		}
		Expect(os.MkdirAll(filepath.Join(demosDir, "empty"), 0o750)).To(Succeed())
	})

	It("resolves names in demos/ and expands a folder to its replays, sorted, not recursive", func() {
		Expect(cmd.ReplayPaths([]string{"leaderboard", "a.sdfz"})).To(Equal([]string{
			filepath.Join(demosDir, "leaderboard", "b.sdfz"),
			filepath.Join(demosDir, "leaderboard", "c.sdfz"),
			filepath.Join(demosDir, "a.sdfz"),
		}))
	})

	It("takes paths of folders and files", func() {
		Expect(cmd.ReplayPaths([]string{filepath.Join(demosDir, "leaderboard", "deeper")})).To(Equal([]string{
			filepath.Join(demosDir, "leaderboard", "deeper", "d.sdfz"),
		}))
	})

	It("returns a replay named twice once", func() {
		Expect(cmd.ReplayPaths([]string{"leaderboard", filepath.Join(demosDir, "leaderboard", "b.sdfz")})).To(Equal(
			[]string{filepath.Join(demosDir, "leaderboard", "b.sdfz"), filepath.Join(demosDir, "leaderboard", "c.sdfz")}))
	})

	It("keeps a missing replay, so that its command fails with its name", func() {
		Expect(cmd.ReplayPaths([]string{"missing.sdfz"})).To(Equal([]string{filepath.Join(demosDir, "missing.sdfz")}))
	})

	It("rejects a folder without replays", func() {
		Expect(cmd.ReplayPaths([]string{"empty"})).Error().To(MatchError(ContainSubstring("no .sdfz files")))
	})
})

package utils_test

import (
	"os"
	"path/filepath"
	"runtime"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/backbone81/bar-replay-analytics-go/internal/utils"
)

// makeBarDataDir creates dir as the BAR launcher does on its first start.
func makeBarDataDir(dir string) {
	Expect(os.MkdirAll(filepath.Join(dir, "launcher-logs"), 0o750)).To(Succeed())
}

// writeUserDirs writes content as user-dirs.dirs into a new XDG_CONFIG_HOME.
func writeUserDirs(content string) {
	configHome := GinkgoT().TempDir()
	GinkgoT().Setenv("XDG_CONFIG_HOME", configHome)
	Expect(os.WriteFile(filepath.Join(configHome, "user-dirs.dirs"), []byte(content), 0o600)).To(Succeed())
}

var _ = Describe("BarDataDir", func() {
	Context("BarDataDirCandidates", func() {
		It("returns the Windows installer's default folder", func() {
			GinkgoT().Setenv("LOCALAPPDATA", "/appdata")

			Expect(utils.BarDataDirCandidates("windows", "/home/user")).
				To(Equal([]string{filepath.Join("/appdata", "Programs", "Beyond-All-Reason", "data")}))
		})

		It("returns none on Windows without LOCALAPPDATA", func() {
			GinkgoT().Setenv("LOCALAPPDATA", "")

			Expect(utils.BarDataDirCandidates("windows", "/home/user")).To(BeEmpty())
		})

		It("returns the Flatpak's and the AppImage's folders on Linux", func() {
			GinkgoT().Setenv("XDG_CONFIG_HOME", GinkgoT().TempDir())
			GinkgoT().Setenv("XDG_STATE_HOME", "")

			Expect(utils.BarDataDirCandidates("linux", "/home/user")).To(Equal([]string{
				filepath.Join("/home/user", ".var", "app", "info.beyondallreason.bar", "data"),
				filepath.Join("/home/user", "Documents", "Beyond All Reason"),
				filepath.Join("/home/user", ".local", "state", "Beyond All Reason"),
			}))
		})

		It("follows XDG_STATE_HOME for the AppImage", func() {
			GinkgoT().Setenv("XDG_CONFIG_HOME", GinkgoT().TempDir())
			GinkgoT().Setenv("XDG_STATE_HOME", "/state")

			Expect(utils.BarDataDirCandidates("linux", "/home/user")).
				To(ContainElement(filepath.Join("/state", "Beyond All Reason")))
		})

		It("follows XDG_DOCUMENTS_DIR relative to $HOME", func() {
			writeUserDirs("# comment\nXDG_DESKTOP_DIR=\"$HOME/Schreibtisch\"\nXDG_DOCUMENTS_DIR=\"$HOME/Dokumente\"\n")

			Expect(utils.BarDataDirCandidates("linux", "/home/user")).
				To(ContainElement(filepath.Join("/home/user", "Dokumente", "Beyond All Reason")))
		})

		It("follows an absolute XDG_DOCUMENTS_DIR", func() {
			writeUserDirs("XDG_DOCUMENTS_DIR=\"/data/docs\"\n")

			Expect(utils.BarDataDirCandidates("linux", "/home/user")).
				To(ContainElement(filepath.Join("/data/docs", "Beyond All Reason")))
		})

		It("uses ~/Documents if user-dirs.dirs has no XDG_DOCUMENTS_DIR", func() {
			writeUserDirs("XDG_DESKTOP_DIR=\"$HOME/Desktop\"\n")

			Expect(utils.BarDataDirCandidates("linux", "/home/user")).
				To(ContainElement(filepath.Join("/home/user", "Documents", "Beyond All Reason")))
		})

		It("returns none on other systems", func() {
			Expect(utils.BarDataDirCandidates("darwin", "/home/user")).To(BeEmpty())
		})
	})

	Context("IsBarDataDir", func() {
		It("accepts a folder with launcher-logs/", func() {
			dir := GinkgoT().TempDir()
			makeBarDataDir(dir)

			Expect(utils.IsBarDataDir(dir)).To(BeTrue())
		})

		It("rejects a folder without launcher-logs/", func() {
			dir := GinkgoT().TempDir()
			Expect(os.MkdirAll(filepath.Join(dir, "engine"), 0o750)).To(Succeed())

			Expect(utils.IsBarDataDir(dir)).To(BeFalse())
		})

		It("rejects a file named launcher-logs", func() {
			dir := GinkgoT().TempDir()
			Expect(os.WriteFile(filepath.Join(dir, "launcher-logs"), nil, 0o600)).To(Succeed())

			Expect(utils.IsBarDataDir(dir)).To(BeFalse())
		})

		It("rejects a folder that does not exist", func() {
			Expect(utils.IsBarDataDir(filepath.Join(GinkgoT().TempDir(), "missing"))).To(BeFalse())
		})
	})

	Context("BarDataDir", func() {
		BeforeEach(func() {
			if runtime.GOOS != "linux" {
				Skip("the candidates depend on the operating system")
			}
		})

		It("returns the first candidate that is a BAR data dir", func() {
			home := GinkgoT().TempDir()
			state := GinkgoT().TempDir()
			GinkgoT().Setenv("HOME", home)
			GinkgoT().Setenv("XDG_CONFIG_HOME", "")
			GinkgoT().Setenv("XDG_STATE_HOME", state)
			appImage := filepath.Join(state, "Beyond All Reason")
			makeBarDataDir(appImage)

			Expect(utils.BarDataDir()).To(Equal(appImage))

			documents := filepath.Join(home, "Documents", "Beyond All Reason")
			makeBarDataDir(documents)

			Expect(utils.BarDataDir()).To(Equal(documents))

			flatpak := filepath.Join(home, ".var", "app", "info.beyondallreason.bar", "data")
			makeBarDataDir(flatpak)

			Expect(utils.BarDataDir()).To(Equal(flatpak))
		})

		It("returns an error if no candidate is a BAR data dir", func() {
			GinkgoT().Setenv("HOME", GinkgoT().TempDir())
			GinkgoT().Setenv("XDG_CONFIG_HOME", "")
			GinkgoT().Setenv("XDG_STATE_HOME", "")

			Expect(utils.BarDataDir()).Error().To(HaveOccurred())
		})
	})
})

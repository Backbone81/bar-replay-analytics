package datadir_test

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/backbone81/bar-replay-analytics-go/internal/datadir"
)

func TestDataDir(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "DataDir Suite")
}

// setenv sets the environment variable name to value, or unsets it if value is nil, and restores its current value
// when the test ends.
func setenv(name string, value *string) {
	previous, wasSet := os.LookupEnv(name)
	DeferCleanup(func() {
		if wasSet {
			Expect(os.Setenv(name, previous)).To(Succeed())
		} else {
			Expect(os.Unsetenv(name)).To(Succeed())
		}
	})
	if value == nil {
		Expect(os.Unsetenv(name)).To(Succeed())
		return
	}
	Expect(os.Setenv(name, *value)).To(Succeed())
}

// mkdir creates the folder dir with its parents.
func mkdir(dir string) {
	Expect(os.MkdirAll(dir, 0o750)).To(Succeed())
}

var _ = Describe("DataDir", func() {
	Context("Candidates", func() {
		It("knows the Windows installer's folder", func() {
			setenv("LOCALAPPDATA", new("/appdata"))

			Expect(datadir.Candidates("windows", "/home")).
				To(Equal([]string{filepath.Join("/appdata", "Programs", "Beyond-All-Reason", "data")}))
		})

		It("knows the Flatpak's and the AppImage's folders on Linux", func() {
			setenv("XDG_STATE_HOME", nil)

			Expect(datadir.Candidates("linux", "/home/user")).To(Equal([]string{
				filepath.Join("/home/user", ".var", "app", "info.beyondallreason.bar", "data"),
				filepath.Join("/home/user", ".local", "state", "Beyond All Reason"),
			}))
		})

		It("follows XDG_STATE_HOME for the AppImage", func() {
			setenv("XDG_STATE_HOME", new("/state"))

			Expect(datadir.Candidates("linux", "/home/user")).
				To(ContainElement(filepath.Join("/state", "Beyond All Reason")))
		})

		It("knows none on other systems", func() {
			Expect(datadir.Candidates("darwin", "/home/user")).To(BeEmpty())
		})
	})

	Context("FindIn", func() {
		It("finds the candidate that holds an engine or demos folder", func() {
			root := GinkgoT().TempDir()
			empty := filepath.Join(root, "empty")
			mkdir(empty)
			bar := filepath.Join(root, "bar")
			mkdir(filepath.Join(bar, "demos"))

			Expect(datadir.FindIn([]string{filepath.Join(root, "missing"), empty, bar})).To(Equal(bar))
		})

		It("fails if no candidate is a data dir", func() {
			root := GinkgoT().TempDir()

			_, err := datadir.FindIn([]string{filepath.Join(root, "missing")})

			Expect(err).To(MatchError(ContainSubstring("--bar-data-dir")))
		})

		It("picks the first if several candidates are data dirs", func() {
			root := GinkgoT().TempDir()
			first := filepath.Join(root, "first")
			mkdir(filepath.Join(first, "engine"))
			second := filepath.Join(root, "second")
			mkdir(filepath.Join(second, "demos"))

			Expect(datadir.FindIn([]string{first, second})).To(Equal(first))
		})
	})
})

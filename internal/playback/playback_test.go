package playback_test

import (
	"bytes"
	"compress/gzip"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/backbone81/bar-replay-analytics-go/internal/newfiles"
	"github.com/backbone81/bar-replay-analytics-go/internal/playback"
	"github.com/backbone81/bar-replay-analytics-go/internal/playback/widget"
)

// writeFile writes content to filePath, creating its folder.
func writeFile(filePath string, content []byte) {
	Expect(os.MkdirAll(filepath.Dir(filePath), 0o750)).To(Succeed())
	Expect(os.WriteFile(filePath, content, 0o600)).To(Succeed())
}

// expectEmbeddedWidget expects widgetDir to hold every file of widget.Template with its embedded content.
func expectEmbeddedWidget(widgetDir string) {
	var filePaths []string
	Expect(fs.WalkDir(widget.Template, ".", func(name string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		filePaths = append(filePaths, name)
		expected, err := fs.ReadFile(widget.Template, name)
		Expect(err).NotTo(HaveOccurred())
		Expect(os.ReadFile(filepath.Join(widgetDir, filepath.FromSlash(name)))).To(Equal(expected))
		return nil
	})).To(Succeed())
	Expect(filePaths).To(ContainElements(playback.AnalyticsWidgetFile, "LICENSE"))
}

// gzipped returns text compressed with gzip.
func gzipped(text string) []byte {
	var buffer bytes.Buffer
	writer := gzip.NewWriter(&buffer)
	_, err := writer.Write([]byte(text))
	Expect(err).NotTo(HaveOccurred())
	Expect(writer.Close()).To(Succeed())
	return buffer.Bytes()
}

var _ = Describe("Playback", func() {
	DescribeTable("EngineFolderName",
		func(engineVersion string, expected string) {
			Expect(playback.EngineFolderName(engineVersion)).To(Equal(expected))
		},
		Entry("Recoil tag", "2026.07.04", "recoil_2026.07.04"),
		Entry("2025.01 release", "2025.01.6", "rel2501.2025.01.6"),
		Entry("Spring engine", "105.1.1-2590-gb9462a0 BAR105", "105.1.1-2590-gb9462a0 bar"),
	)

	DescribeTable("NormalizeMapName",
		func(name string, expected string) {
			Expect(playback.NormalizeMapName(name)).To(Equal(expected))
		},
		Entry("map name", "Argent Strata 1.1", "argent_strata_1_1"),
		Entry("archive name", "argent_strata_1.1", "argent_strata_1_1"),
		Entry("apostrophe", "Devil's Postpiles 1.1.1", "devil_s_postpiles_1_1_1"),
		Entry("leading and trailing punctuation", "_Map v2_", "map_v2"),
	)

	Context("FindMapArchive", func() {
		It("finds the archive whose file name matches the map name", func() {
			mapsDir := GinkgoT().TempDir()
			writeFile(filepath.Join(mapsDir, "other_map_1.0.sd7"), nil)
			writeFile(filepath.Join(mapsDir, "argent_strata_1.1.sd7"), nil)

			archive, found, err := playback.FindMapArchive(mapsDir, "Argent Strata 1.1")

			Expect(err).NotTo(HaveOccurred())
			Expect(found).To(BeTrue())
			Expect(archive).To(Equal(filepath.Join(mapsDir, "argent_strata_1.1.sd7")))
		})

		It("ignores files which are no map archives", func() {
			mapsDir := GinkgoT().TempDir()
			writeFile(filepath.Join(mapsDir, "argent_strata_1.1.txt"), nil)

			_, found, err := playback.FindMapArchive(mapsDir, "Argent Strata 1.1")

			Expect(err).NotTo(HaveOccurred())
			Expect(found).To(BeFalse())
		})

		It("finds nothing in a missing folder", func() {
			_, found, err := playback.FindMapArchive(filepath.Join(GinkgoT().TempDir(), "maps"), "Argent Strata 1.1")

			Expect(err).NotTo(HaveOccurred())
			Expect(found).To(BeFalse())
		})
	})

	Context("FindGamePackage", func() {
		const gameType = "Beyond All Reason test-31720-0f97aa5"

		It("finds the package of the game", func() {
			dataDir := GinkgoT().TempDir()
			writeFile(filepath.Join(dataDir, "rapid", "repos-cdn.beyondallreason.dev", "byar", "versions.gz"), gzipped(
				"byar:test,1111,,Beyond All Reason test-31719-aaaaaaa\n"+
					"byar:test,2222,,"+gameType+"\n"))
			writeFile(filepath.Join(dataDir, "packages", "1111.sdp"), nil)
			writeFile(filepath.Join(dataDir, "packages", "2222.sdp"), nil)

			gamePackage, found, err := playback.FindGamePackage(dataDir, gameType)

			Expect(err).NotTo(HaveOccurred())
			Expect(found).To(BeTrue())
			Expect(gamePackage).To(Equal(filepath.Join(dataDir, "packages", "2222.sdp")))
		})

		It("does not count a listed package whose file is missing", func() {
			dataDir := GinkgoT().TempDir()
			writeFile(filepath.Join(dataDir, "rapid", "host", "byar", "versions.gz"),
				gzipped("byar:test,2222,,"+gameType+"\n"))

			_, found, err := playback.FindGamePackage(dataDir, gameType)

			Expect(err).NotTo(HaveOccurred())
			Expect(found).To(BeFalse())
		})

		It("fails on a versions file which is not gzipped", func() {
			dataDir := GinkgoT().TempDir()
			writeFile(filepath.Join(dataDir, "rapid", "host", "byar", "versions.gz"), []byte("byar:test,2222,,x\n"))

			Expect(playback.FindGamePackage(dataDir, gameType)).Error().To(HaveOccurred())
		})
	})

	Context("FindWidgetDir", func() {
		It("finds the folder of a widget one level deep", func() {
			widgets := GinkgoT().TempDir()
			writeFile(filepath.Join(widgets, "bar-replay-analytics", playback.AnalyticsWidgetFile), nil)

			dir, found, err := playback.FindWidgetDir(widgets, playback.AnalyticsWidgetFile)

			Expect(err).NotTo(HaveOccurred())
			Expect(found).To(BeTrue())
			Expect(dir).To(Equal(filepath.Join(widgets, "bar-replay-analytics")))
		})

		It("does not find a widget deeper down", func() {
			widgets := GinkgoT().TempDir()
			writeFile(filepath.Join(widgets, "a", "b", playback.AnalyticsWidgetFile), nil)

			_, found, err := playback.FindWidgetDir(widgets, playback.AnalyticsWidgetFile)

			Expect(err).NotTo(HaveOccurred())
			Expect(found).To(BeFalse())
		})
	})

	Context("NewPlayer", func() {
		It("fails if the analytics widget is not installed", func() {
			Expect(playback.NewPlayer(GinkgoT().TempDir())).Error().To(HaveOccurred())
		})

		It("finds the analytics widget and its events folder", func() {
			dataDir := GinkgoT().TempDir()
			writeFile(filepath.Join(dataDir, "LuaUI", "Widgets", "analytics", playback.AnalyticsWidgetFile), nil)

			player, err := playback.NewPlayer(dataDir)

			Expect(err).NotTo(HaveOccurred())
			Expect(player.AnalyticsWidgetDir()).To(Equal(filepath.Join(dataDir, "LuaUI", "Widgets", "analytics")))
			Expect(player.EventsDir()).To(Equal(filepath.Join(dataDir, "LuaUI", "Widgets", "analytics", "events")))
		})
	})

	Context("InstallAnalyticsWidget", func() {
		It("installs the widget and its license into a new folder", func() {
			dataDir := GinkgoT().TempDir()

			widgetDir, created, err := playback.InstallAnalyticsWidget(dataDir)

			Expect(err).NotTo(HaveOccurred())
			Expect(widgetDir).To(Equal(filepath.Join(dataDir, "LuaUI", "Widgets", "bar-replay-analytics")))
			Expect(created).To(ConsistOf(
				filepath.Join(widgetDir, playback.AnalyticsWidgetFile),
				filepath.Join(widgetDir, "LICENSE"),
			))
			expectEmbeddedWidget(widgetDir)
			Expect(playback.NewPlayer(dataDir)).Error().NotTo(HaveOccurred())
		})

		It("adds missing files to the folder of an installed widget without overwriting it", func() {
			dataDir := GinkgoT().TempDir()
			installed := filepath.Join(dataDir, "LuaUI", "Widgets", "analytics")
			widgetFile := filepath.Join(installed, playback.AnalyticsWidgetFile)
			writeFile(widgetFile, []byte("-- old version"))

			widgetDir, created, err := playback.InstallAnalyticsWidget(dataDir)

			var existErr *newfiles.ExistError
			Expect(errors.As(err, &existErr)).To(BeTrue())
			Expect(existErr.FilePaths).To(Equal([]string{widgetFile}))
			Expect(widgetDir).To(Equal(installed))
			Expect(created).To(Equal([]string{filepath.Join(installed, "LICENSE")}))
			Expect(os.ReadFile(widgetFile)).To(Equal([]byte("-- old version")))
		})
	})

	DescribeTable("LastFrameIn",
		func(data string, expected int) {
			Expect(playback.LastFrameIn([]byte(data))).To(Equal(expected))
		},
		Entry("no events yet", `{"events":[`+"\n", 0),
		Entry("first event", `{"events":[`+"\n"+`{"frame":3,"unitCreated":{}}`, 3),
		Entry("events separated by commas", `{"frame":3,"a":{}},`+"\n"+`{"frame":7,"b":{}}`, 7),
		Entry("last line still being written", `{"frame":3,"a":{}}`+",\n"+`{"frame":9,"b":{"unit`, 3),
		Entry("first line cut off", `:1}}`+",\n"+`{"frame":5,"a":{}}`, 5),
	)

	DescribeTable("ParallelFor",
		func(cpus int, availableMemory int64, expected int) {
			Expect(playback.ParallelFor(cpus, availableMemory)).To(Equal(expected))
		},
		Entry("limited by memory", 32, int64(30<<30), 5),
		Entry("limited by CPUs", 8, int64(64<<30), 4),
		Entry("memory not known", 32, int64(0), 16),
		Entry("at least one with little memory", 32, int64(2<<30), 1),
		Entry("at least one with one CPU", 1, int64(64<<30), 1),
	)

	It("finds a default number of parallel playbacks", func() {
		Expect(playback.DefaultParallel()).To(BeNumerically(">=", 1))
	})

	Context("Progress", func() {
		It("tells the percentage of the game", func() {
			progress := playback.Progress{Replay: &playback.Replay{GameTime: 100 * time.Second}, Frame: 25 * 30}

			percent, known := progress.Percent()

			Expect(known).To(BeTrue())
			Expect(percent).To(BeNumerically("~", 25))
		})

		It("does not know the percentage without the length of the game", func() {
			progress := playback.Progress{Replay: &playback.Replay{}, Frame: 25 * 30}

			_, known := progress.Percent()

			Expect(known).To(BeFalse())
		})
	})
})

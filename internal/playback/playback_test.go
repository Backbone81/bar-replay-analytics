package playback_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/backbone81/bar-replay-analytics-go/internal/playback"
	"github.com/backbone81/bar-replay-analytics-go/internal/playback/widget"
)

// analyticsWidgetFile is the file name of the analytics widget.
const analyticsWidgetFile = widget.FileName

// discardLog is the logger for the functions under test; their log lines are not tested here.
var discardLog = slog.New(slog.DiscardHandler)

// writeFile writes content to filePath, creating its folder.
func writeFile(filePath string, content []byte) {
	Expect(os.MkdirAll(filepath.Dir(filePath), 0o750)).To(Succeed())
	Expect(os.WriteFile(filePath, content, 0o600)).To(Succeed())
}

// templateFiles returns the paths of the files of widget.Template, joined to dir.
func templateFiles(dir string) []string {
	var filePaths []string
	Expect(fs.WalkDir(widget.Template, ".", func(name string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		filePaths = append(filePaths, filepath.Join(dir, filepath.FromSlash(name)))
		return nil
	})).To(Succeed())
	return filePaths
}

// expectEmbeddedWidget expects widgetDir to hold every file of widget.Template with its embedded content.
func expectEmbeddedWidget(widgetDir string) {
	filePaths := templateFiles(widgetDir)
	Expect(filePaths).To(ContainElements(
		filepath.Join(widgetDir, analyticsWidgetFile),
		filepath.Join(widgetDir, "README.md"),
		filepath.Join(widgetDir, "LICENSE"),
	))
	for _, filePath := range filePaths {
		name, err := filepath.Rel(widgetDir, filePath)
		Expect(err).NotTo(HaveOccurred())
		expected, err := fs.ReadFile(widget.Template, filepath.ToSlash(name))
		Expect(err).NotTo(HaveOccurred())
		Expect(os.ReadFile(filePath)).To(Equal(expected))
	}
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
			writeFile(filepath.Join(widgets, "bar-replay-analytics", analyticsWidgetFile), nil)

			dir, found, err := playback.FindWidgetDir(widgets, analyticsWidgetFile)

			Expect(err).NotTo(HaveOccurred())
			Expect(found).To(BeTrue())
			Expect(dir).To(Equal(filepath.Join(widgets, "bar-replay-analytics")))
		})

		It("does not find a widget deeper down", func() {
			widgets := GinkgoT().TempDir()
			writeFile(filepath.Join(widgets, "a", "b", analyticsWidgetFile), nil)

			_, found, err := playback.FindWidgetDir(widgets, analyticsWidgetFile)

			Expect(err).NotTo(HaveOccurred())
			Expect(found).To(BeFalse())
		})
	})

	Context("EventsFilePath and LogFilePath", func() {
		It("name the files after the replay, next to it", func() {
			replayFile := filepath.Join("demos", "2026-10-24_13-48-21-286_Argent Strata 1.1_2026.07.04.sdfz")

			Expect(playback.EventsFilePath(replayFile)).To(Equal(filepath.Join("demos",
				"2026-10-24_13-48-21-286_Argent Strata 1.1_2026.07.04_events.json")))
			Expect(playback.LogFilePath(replayFile)).To(Equal(filepath.Join("demos",
				"2026-10-24_13-48-21-286_Argent Strata 1.1_2026.07.04.log")))
		})
	})

	Context("Unpack7z", func() {
		It("unpacks folders and files with their permissions", func() {
			destination := filepath.Join(GinkgoT().TempDir(), "engine")

			Expect(playback.Unpack7z(context.Background(), filepath.Join("testdata", "engine.7z"), destination)).
				To(Succeed())

			Expect(os.ReadFile(filepath.Join(destination, "spring-headless"))).To(Equal([]byte("#!/bin/sh\n")))
			Expect(os.ReadFile(filepath.Join(destination, "AI", "config.json"))).To(Equal([]byte("data\n")))
			info, err := os.Stat(filepath.Join(destination, "spring-headless"))
			Expect(err).NotTo(HaveOccurred())
			Expect(info.Mode().Perm() & 0o111).NotTo(BeZero())
		})

		It("stops when the context is cancelled", func() {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			Expect(playback.Unpack7z(ctx, filepath.Join("testdata", "engine.7z"), GinkgoT().TempDir())).
				To(MatchError(context.Canceled))
		})
	})

	Context("MakeWriteDir", func() {
		It("writes the embedded analytics widget, which writes its events into the write dir", func() {
			dataDir := GinkgoT().TempDir()
			player, err := playback.NewPlayer(dataDir)
			Expect(err).NotTo(HaveOccurred())

			writeDir, err := player.MakeWriteDir(discardLog)

			Expect(err).NotTo(HaveOccurred())
			DeferCleanup(os.RemoveAll, writeDir)
			widgetDir := filepath.Join(writeDir, "LuaUI", "Widgets", "bar-replay-analytics")
			expectEmbeddedWidget(widgetDir)
			Expect(filepath.Join(widgetDir, "events")).NotTo(BeAnExistingFile())
			Expect(player.WriteDirEventsFile(writeDir, &playback.Replay{Name: "game"})).
				To(Equal(filepath.Join(widgetDir, "events", "game_events.json")))
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

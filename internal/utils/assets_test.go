package utils_test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/backbone81/bar-replay-analytics-go/internal/utils"
)

var _ = Describe("Assets", func() {
	Context("ValidateEngineVersion", func() {
		It("accepts a release tag", func() {
			Expect(utils.ValidateEngineVersion(testEngineVersion)).To(Succeed())
		})

		DescribeTable("rejects what is not a release tag",
			func(engineVersion string) {
				Expect(utils.ValidateEngineVersion(engineVersion)).ToNot(Succeed())
			},
			Entry("a path out of the engine folder", "../../2026.07.04"),
			Entry("URL syntax", "2026.07.04&category=other"),
			Entry("a local build", "local-build"),
			Entry("a tag with extra text", "2026.07.04-rc1"),
			Entry("an empty version", ""),
		)
	})

	Context("EngineVersions", func() {
		It("finds nothing in an empty data dir", func() {
			Expect(utils.EngineVersions(newDataDir())).To(BeEmpty())
		})

		It("finds an engine with spring-headless and pr-downloader", func() {
			dataDir := newDataDir()
			addEngineFiles(dataDir, "recoil_"+testEngineVersion, "spring-headless", "pr-downloader")

			Expect(utils.EngineVersions(dataDir)).To(HaveExactElements(testEngineVersion))
		})

		It("finds an engine with the Windows executables", func() {
			dataDir := newDataDir()
			addEngineFiles(dataDir, "recoil_"+testEngineVersion, "spring-headless.exe", "pr-downloader.exe")

			Expect(utils.EngineVersions(dataDir)).To(HaveExactElements(testEngineVersion))
		})

		It("returns the engines sorted", func() {
			dataDir := newDataDir()
			addEngineFiles(dataDir, "recoil_2026.08.01", "spring-headless", "pr-downloader")
			addEngineFiles(dataDir, "recoil_"+testEngineVersion, "spring-headless", "pr-downloader")

			Expect(utils.EngineVersions(dataDir)).To(HaveExactElements(testEngineVersion, "2026.08.01"))
		})

		It("does not count an engine without pr-downloader", func() {
			dataDir := newDataDir()
			addEngineFiles(dataDir, "recoil_"+testEngineVersion, "spring-headless")

			Expect(utils.EngineVersions(dataDir)).To(BeEmpty())
		})

		It("does not count an engine without spring-headless", func() {
			dataDir := newDataDir()
			addEngineFiles(dataDir, "recoil_"+testEngineVersion, "pr-downloader")

			Expect(utils.EngineVersions(dataDir)).To(BeEmpty())
		})

		It("does not count an engine whose spring-headless is a folder", func() {
			dataDir := newDataDir()
			addEngineFiles(dataDir, "recoil_"+testEngineVersion, "pr-downloader")
			Expect(os.Mkdir(utils.SpringHeadlessFilePath(dataDir, testEngineVersion), 0o750)).To(Succeed())

			Expect(utils.EngineVersions(dataDir)).To(BeEmpty())
		})

		It("does not count an engine folder that is not named after a release tag", func() {
			dataDir := newDataDir()
			addEngineFiles(dataDir, "recoil_local-build", "spring-headless", "pr-downloader")

			Expect(utils.EngineVersions(dataDir)).To(BeEmpty())
		})

		It("does not count a file named like an engine folder", func() {
			dataDir := newDataDir()
			writeFile(utils.EngineVersionDir(dataDir, testEngineVersion), "")

			Expect(utils.EngineVersions(dataDir)).To(BeEmpty())
		})
	})

	Context("Maps", func() {
		It("finds nothing in an empty data dir", func() {
			Expect(utils.Maps(newDataDir())).To(BeEmpty())
		})

		It("finds a map in a .sd7 archive by its normalized name", func() {
			dataDir := newDataDir()
			addMapFile(dataDir, testMapFileName)

			Expect(utils.Maps(dataDir)).To(HaveExactElements(utils.NormalizeMapName(testMapName)))
		})

		It("finds a map in a .sdz archive whose file name has upper case letters", func() {
			dataDir := newDataDir()
			addMapFile(dataDir, "Supreme_Isthmus_V1.8.SDZ")

			Expect(utils.Maps(dataDir)).To(HaveExactElements(utils.NormalizeMapName("Supreme Isthmus v1.8")))
		})

		It("does not count files which are no map archives", func() {
			dataDir := newDataDir()
			addMapFile(dataDir, "argent_strata_1.1.txt")

			Expect(utils.Maps(dataDir)).To(BeEmpty())
		})

		It("does not count folders", func() {
			dataDir := newDataDir()
			Expect(os.MkdirAll(filepath.Join(utils.MapsDir(dataDir), testMapFileName), 0o750)).To(Succeed())

			Expect(utils.Maps(dataDir)).To(BeEmpty())
		})
	})

	Context("Games", func() {
		It("finds nothing in an empty data dir", func() {
			Expect(utils.Games(newDataDir())).To(BeEmpty())
		})

		It("finds a game whose package is there", func() {
			dataDir := newDataDir()
			addGame(dataDir, testGamePackage, testGameType)

			Expect(utils.Games(dataDir)).To(HaveExactElements(testGameType))
		})

		It("does not count a listed game whose package is missing", func() {
			dataDir := newDataDir()
			writeRapidVersions(dataDir,
				"byar:test,"+testGamePackage+",,"+testGameType,
				"byar:test,fedcba9876543210fedcba9876543210,,Beyond All Reason test-31000-1234567",
			)
			addPackage(dataDir, testGamePackage)

			Expect(utils.Games(dataDir)).To(HaveExactElements(testGameType))
		})

		It("finds no game without packages/", func() {
			dataDir := newDataDir()
			writeRapidVersions(dataDir, "byar:test,"+testGamePackage+",,"+testGameType)

			Expect(utils.Games(dataDir)).To(BeEmpty())
		})

		It("skips lines with too few fields", func() {
			dataDir := newDataDir()
			writeRapidVersions(dataDir, "byar:test,"+testGamePackage, "byar:test,"+testGamePackage+",,"+testGameType)
			addPackage(dataDir, testGamePackage)

			Expect(utils.Games(dataDir)).To(HaveExactElements(testGameType))
		})

		It("fails on a versions file that is not gzipped", func() {
			dataDir := newDataDir()
			writeFile(rapidVersionsFile(dataDir), "byar:test,"+testGamePackage+",,"+testGameType+"\n")

			Expect(utils.Games(dataDir)).Error().To(HaveOccurred())
		})
	})

	DescribeTable("NormalizeMapName",
		func(mapName string, expected string) {
			Expect(utils.NormalizeMapName(mapName)).To(Equal(expected))
		},
		Entry("lower case with underscores", "Argent Strata 1.1", "argent_strata_1_1"),
		Entry("an archive file name", "argent_strata_1.1", "argent_strata_1_1"),
		Entry("other characters", "Tabula-v7 (remake)", "tabula_v7__remake_"),
		Entry("non-ASCII letters, byte by byte", "Glacé", "glac__"),
	)
})

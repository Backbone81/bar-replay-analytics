package replayplayer_test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/backbone81/bar-replay-analytics-go/internal/replayplayer"
)

var _ = Describe("Inventory", func() {
	It("finds nothing in an empty data dir", func() {
		inventory, err := replayplayer.NewInventory(newDataDir())

		Expect(err).NotTo(HaveOccurred())
		Expect(inventory.EngineVersionAvailable(testEngineVersion)).To(BeFalse())
		Expect(inventory.MapAvailable(testMapName)).To(BeFalse())
		Expect(inventory.GameAvailable(testGameType)).To(BeFalse())
	})

	Context("engines", func() {
		It("finds an engine with spring-headless and pr-downloader", func() {
			dataDir := newDataDir()
			addEngineFiles(dataDir, "recoil_"+testEngineVersion, "spring-headless", "pr-downloader")

			inventory, err := replayplayer.NewInventory(dataDir)

			Expect(err).NotTo(HaveOccurred())
			Expect(inventory.EngineVersionAvailable(testEngineVersion)).To(BeTrue())
		})

		It("finds an engine with the Windows executables", func() {
			dataDir := newDataDir()
			addEngineFiles(dataDir, "recoil_"+testEngineVersion, "spring-headless.exe", "pr-downloader.exe")

			inventory, err := replayplayer.NewInventory(dataDir)

			Expect(err).NotTo(HaveOccurred())
			Expect(inventory.EngineVersionAvailable(testEngineVersion)).To(BeTrue())
		})

		It("does not count an engine without pr-downloader", func() {
			dataDir := newDataDir()
			addEngineFiles(dataDir, "recoil_"+testEngineVersion, "spring-headless")

			inventory, err := replayplayer.NewInventory(dataDir)

			Expect(err).NotTo(HaveOccurred())
			Expect(inventory.EngineVersionAvailable(testEngineVersion)).To(BeFalse())
		})

		It("does not count an engine without spring-headless", func() {
			dataDir := newDataDir()
			addEngineFiles(dataDir, "recoil_"+testEngineVersion, "pr-downloader")

			inventory, err := replayplayer.NewInventory(dataDir)

			Expect(err).NotTo(HaveOccurred())
			Expect(inventory.EngineVersionAvailable(testEngineVersion)).To(BeFalse())
		})

		It("does not count an engine whose spring-headless is a folder", func() {
			dataDir := newDataDir()
			addEngineFiles(dataDir, "recoil_"+testEngineVersion, "pr-downloader")
			Expect(os.Mkdir(filepath.Join(engineDir(dataDir, testEngineVersion), "spring-headless"), 0o750)).
				To(Succeed())

			inventory, err := replayplayer.NewInventory(dataDir)

			Expect(err).NotTo(HaveOccurred())
			Expect(inventory.EngineVersionAvailable(testEngineVersion)).To(BeFalse())
		})

		It("does not count an engine folder that is not named after a release tag", func() {
			dataDir := newDataDir()
			addEngineFiles(dataDir, "recoil_local-build", "spring-headless", "pr-downloader")

			inventory, err := replayplayer.NewInventory(dataDir)

			Expect(err).NotTo(HaveOccurred())
			Expect(inventory.EngineVersionAvailable("local-build")).To(BeFalse())
		})

		It("does not count a file named like an engine folder", func() {
			dataDir := newDataDir()
			writeFile(engineDir(dataDir, testEngineVersion), "")

			inventory, err := replayplayer.NewInventory(dataDir)

			Expect(err).NotTo(HaveOccurred())
			Expect(inventory.EngineVersionAvailable(testEngineVersion)).To(BeFalse())
		})
	})

	Context("maps", func() {
		It("finds a map by its name in a .sd7 archive", func() {
			dataDir := newDataDir()
			addMapFile(dataDir, testMapFileName)

			inventory, err := replayplayer.NewInventory(dataDir)

			Expect(err).NotTo(HaveOccurred())
			Expect(inventory.MapAvailable(testMapName)).To(BeTrue())
		})

		It("finds a map in a .sdz archive whose file name has upper case letters", func() {
			dataDir := newDataDir()
			addMapFile(dataDir, "Supreme_Isthmus_V1.8.SDZ")

			inventory, err := replayplayer.NewInventory(dataDir)

			Expect(err).NotTo(HaveOccurred())
			Expect(inventory.MapAvailable("Supreme Isthmus v1.8")).To(BeTrue())
		})

		It("does not count files which are no map archives", func() {
			dataDir := newDataDir()
			addMapFile(dataDir, "argent_strata_1.1.txt")

			inventory, err := replayplayer.NewInventory(dataDir)

			Expect(err).NotTo(HaveOccurred())
			Expect(inventory.MapAvailable(testMapName)).To(BeFalse())
		})

		It("does not count folders", func() {
			dataDir := newDataDir()
			Expect(os.MkdirAll(filepath.Join(dataDir, "maps", testMapFileName), 0o750)).To(Succeed())

			inventory, err := replayplayer.NewInventory(dataDir)

			Expect(err).NotTo(HaveOccurred())
			Expect(inventory.MapAvailable(testMapName)).To(BeFalse())
		})
	})

	Context("games", func() {
		It("finds a game whose package is there", func() {
			dataDir := newDataDir()
			addGame(dataDir, testGamePackage, testGameType)

			inventory, err := replayplayer.NewInventory(dataDir)

			Expect(err).NotTo(HaveOccurred())
			Expect(inventory.GameAvailable(testGameType)).To(BeTrue())
		})

		It("does not count a listed game whose package is missing", func() {
			dataDir := newDataDir()
			writeRapidVersions(dataDir,
				"byar:test,"+testGamePackage+",,"+testGameType,
				"byar:test,fedcba9876543210fedcba9876543210,,Beyond All Reason test-31000-1234567",
			)
			addPackage(dataDir, testGamePackage)

			inventory, err := replayplayer.NewInventory(dataDir)

			Expect(err).NotTo(HaveOccurred())
			Expect(inventory.GameAvailable(testGameType)).To(BeTrue())
			Expect(inventory.GameAvailable("Beyond All Reason test-31000-1234567")).To(BeFalse())
		})

		It("finds no game without packages/", func() {
			dataDir := newDataDir()
			writeRapidVersions(dataDir, "byar:test,"+testGamePackage+",,"+testGameType)

			inventory, err := replayplayer.NewInventory(dataDir)

			Expect(err).NotTo(HaveOccurred())
			Expect(inventory.GameAvailable(testGameType)).To(BeFalse())
		})

		It("skips lines with too few fields", func() {
			dataDir := newDataDir()
			writeRapidVersions(dataDir, "byar:test,"+testGamePackage, "byar:test,"+testGamePackage+",,"+testGameType)
			addPackage(dataDir, testGamePackage)

			inventory, err := replayplayer.NewInventory(dataDir)

			Expect(err).NotTo(HaveOccurred())
			Expect(inventory.GameAvailable(testGameType)).To(BeTrue())
		})

		It("fails on a versions file that is not gzipped", func() {
			dataDir := newDataDir()
			writeFile(rapidVersionsFile(dataDir), "byar:test,"+testGamePackage+",,"+testGameType+"\n")

			Expect(replayplayer.NewInventory(dataDir)).Error().To(HaveOccurred())
		})
	})

	Context("Add", func() {
		It("AddEngineVersion makes an engine available", func() {
			inventory, err := replayplayer.NewInventory(newDataDir())
			Expect(err).NotTo(HaveOccurred())

			inventory.AddEngineVersion(testEngineVersion)
			inventory.AddEngineVersion(testEngineVersion)

			Expect(inventory.EngineVersionAvailable(testEngineVersion)).To(BeTrue())
			Expect(inventory.EngineVersionAvailable("2026.01.01")).To(BeFalse())
		})

		It("AddMap makes a map available by its name", func() {
			inventory, err := replayplayer.NewInventory(newDataDir())
			Expect(err).NotTo(HaveOccurred())

			inventory.AddMap(testMapName)
			inventory.AddMap("Supreme Isthmus v1.8")

			Expect(inventory.MapAvailable(testMapName)).To(BeTrue())
			Expect(inventory.MapAvailable("supreme isthmus v1.8")).To(BeTrue())
		})

		It("AddGame makes a game available", func() {
			inventory, err := replayplayer.NewInventory(newDataDir())
			Expect(err).NotTo(HaveOccurred())

			inventory.AddGame(testGameType)

			Expect(inventory.GameAvailable(testGameType)).To(BeTrue())
		})
	})

	It("Refresh finds what was added to the data dir since", func() {
		dataDir := newDataDir()
		inventory, err := replayplayer.NewInventory(dataDir)
		Expect(err).NotTo(HaveOccurred())
		addMapFile(dataDir, testMapFileName)
		Expect(inventory.MapAvailable(testMapName)).To(BeFalse())

		Expect(inventory.Refresh()).To(Succeed())

		Expect(inventory.MapAvailable(testMapName)).To(BeTrue())
	})

	DescribeTable("NormalizeMapName",
		func(mapName string, expected string) {
			Expect(replayplayer.NormalizeMapName(mapName)).To(Equal(expected))
		},
		Entry("lower case with underscores", "Argent Strata 1.1", "argent_strata_1_1"),
		Entry("an archive file name", "argent_strata_1.1", "argent_strata_1_1"),
		Entry("other characters", "Tabula-v7 (remake)", "tabula_v7__remake_"),
		Entry("non-ASCII letters, byte by byte", "Glacé", "glac__"),
	)
})

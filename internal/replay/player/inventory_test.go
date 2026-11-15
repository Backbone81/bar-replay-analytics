package player_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	replayplayer "github.com/backbone81/bar-replay-analytics/internal/replay/player"
)

var _ = Describe("Inventory", func() {
	It("finds nothing in an empty data dir", func() {
		inventory, err := replayplayer.NewInventory(newDataDir())

		Expect(err).NotTo(HaveOccurred())
		Expect(inventory.EngineVersionAvailable(testEngineVersion)).To(BeFalse())
		Expect(inventory.MapAvailable(testMapName)).To(BeFalse())
		Expect(inventory.GameAvailable(testGameType)).To(BeFalse())
	})

	It("finds the engine, map and game in the data dir", func() {
		inventory, err := replayplayer.NewInventory(newDataDirWithAssets())

		Expect(err).NotTo(HaveOccurred())
		Expect(inventory.EngineVersionAvailable(testEngineVersion)).To(BeTrue())
		Expect(inventory.MapAvailable(testMapName)).To(BeTrue())
		Expect(inventory.MapAvailable("argent strata 1.1")).To(BeTrue())
		Expect(inventory.GameAvailable(testGameType)).To(BeTrue())
	})

	It("fails if the data dir cannot be read", func() {
		dataDir := newDataDir()
		writeFile(rapidVersionsFile(dataDir), "not gzipped")

		Expect(replayplayer.NewInventory(dataDir)).Error().To(HaveOccurred())
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
})

package player

import (
	"path/filepath"
	"slices"

	"github.com/backbone81/bar-replay-analytics-go/internal/utils"
)

// Inventory lists the engines, maps and games in a data dir, so a playback can tell without asking the network or
// starting the engine whether something must be downloaded. It is not safe for concurrent use.
type Inventory struct {
	dataDir string

	// The lists are sorted for binary search.
	engineVersions []string // engine tags, e.g. 2026.07.04
	maps           []string // normalized map names (see utils.NormalizeMapName)
	games          []string // rapid names, e.g. "Beyond All Reason test-31720-0f97aa5"
}

// NewInventory returns the inventory of the data dir dataDir.
func NewInventory(dataDir string) (*Inventory, error) {
	dataDir, err := filepath.Abs(dataDir)
	if err != nil {
		return nil, err
	}

	result := Inventory{
		dataDir: dataDir,
	}
	if err := result.Refresh(); err != nil {
		return nil, err
	}
	return &result, nil
}

// EngineVersionAvailable returns whether the engine engineVersion is in the data dir.
func (i *Inventory) EngineVersionAvailable(engineVersion string) bool {
	_, found := slices.BinarySearch(i.engineVersions, engineVersion)
	return found
}

// AddEngineVersion adds the engine engineVersion after it was downloaded, without scanning the data dir again.
func (i *Inventory) AddEngineVersion(engineVersion string) {
	if i.EngineVersionAvailable(engineVersion) {
		return
	}
	i.engineVersions = append(i.engineVersions, engineVersion)
	slices.Sort(i.engineVersions)
}

// MapAvailable returns whether the map mapName (a start script's mapname) is in the data dir.
func (i *Inventory) MapAvailable(mapName string) bool {
	mapName = utils.NormalizeMapName(mapName)
	_, found := slices.BinarySearch(i.maps, mapName)
	return found
}

// AddMap adds the map mapName after it was downloaded, without scanning the data dir again.
func (i *Inventory) AddMap(mapName string) {
	if i.MapAvailable(mapName) {
		return
	}
	i.maps = append(i.maps, utils.NormalizeMapName(mapName))
	slices.Sort(i.maps)
}

// GameAvailable returns whether the game (a start script's gametype) is in the data dir.
func (i *Inventory) GameAvailable(game string) bool {
	_, found := slices.BinarySearch(i.games, game)
	return found
}

// AddGame adds the game after it was downloaded, without scanning the data dir again.
func (i *Inventory) AddGame(game string) {
	if i.GameAvailable(game) {
		return
	}
	i.games = append(i.games, game)
	slices.Sort(i.games)
}

// Refresh scans the data dir again.
func (i *Inventory) Refresh() error {
	engineVersions, err := utils.EngineVersions(i.dataDir)
	if err != nil {
		return err
	}
	i.engineVersions = engineVersions

	maps, err := utils.Maps(i.dataDir)
	if err != nil {
		return err
	}
	i.maps = maps

	games, err := utils.Games(i.dataDir)
	if err != nil {
		return err
	}
	i.games = games

	return nil
}

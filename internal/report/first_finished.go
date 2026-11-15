package report

import (
	"github.com/backbone81/bar-replay-analytics/internal/events"
)

// FirstFinishedEntry is the first unit or building of one unit def a player built.
type FirstFinishedEntry struct {
	// Time is when the unit was finished.
	Time Duration `json:"time"`

	// Name is the name the BAR UI shows, like "Fusion Reactor".
	Name string `json:"name"`
}

// newFirstFinished prepares the first finished units of each of numPlayers players, by player ID: for every unit def
// a player built (see builtUnits) over the whole game, the first unit of it, in the order they were finished.
func newFirstFinished(eventsFile *events.File, numPlayers int) [][]FirstFinishedEntry {
	firstFinished := make([][]FirstFinishedEntry, numPlayers)
	seen := make([]map[int]bool, numPlayers)
	for i := range firstFinished {
		firstFinished[i] = []FirstFinishedEntry{}
		seen[i] = make(map[int]bool)
	}

	for unit := range builtUnits(eventsFile, numPlayers) {
		if seen[unit.playerID][unit.unitDefID] {
			continue
		}
		seen[unit.playerID][unit.unitDefID] = true
		firstFinished[unit.playerID] = append(firstFinished[unit.playerID], FirstFinishedEntry{
			Time: unit.time(),
			Name: eventsFile.UnitDefs[unit.unitDefID].TranslatedHumanName,
		})
	}
	return firstFinished
}

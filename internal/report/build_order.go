package report

import (
	"github.com/backbone81/bar-replay-analytics-go/internal/events"
)

// BuildOrderEntry is one entry of a build order: a unit or building a player built.
type BuildOrderEntry struct {
	// Time is when the unit was finished.
	Time Duration `json:"time"`

	// Name is the name the BAR UI shows, like "Bot Lab".
	Name string `json:"name"`
}

// newOpeningBuildOrders prepares the build order of the opening of each of numPlayers players, by player ID: every unit
// and building a player's builders and factories built and finished before Config.OpeningLength, in the order they were
// finished.
func newOpeningBuildOrders(eventsFile *events.File, numPlayers int, config Config) [][]BuildOrderEntry {
	buildOrders := make([][]BuildOrderEntry, numPlayers)
	for i := range buildOrders {
		buildOrders[i] = []BuildOrderEntry{}
	}

	openingEnd := frames(config.OpeningLength)
	built := make(map[int]bool)
	for _, event := range eventsFile.Events {
		if float64(event.Frame) >= openingEnd {
			break
		}
		switch {
		case event.UnitCreated != nil:
			// Unit IDs are reused once a unit died, so a unit without builder resets its ID.
			built[event.UnitCreated.UnitID] = event.UnitCreated.BuilderID != nil
		case event.UnitFinished != nil:
			finished := event.UnitFinished
			if !built[finished.UnitID] || finished.UnitTeam < 0 || finished.UnitTeam >= numPlayers {
				continue
			}
			unitDef := eventsFile.UnitDefs[finished.UnitDefID]
			buildOrders[finished.UnitTeam] = append(buildOrders[finished.UnitTeam], BuildOrderEntry{
				// The engine counts frames in an int32.
				Time: frameDuration(int32(event.Frame)), //nolint:gosec
				Name: unitDef.TranslatedHumanName,
			})
		}
	}
	return buildOrders
}

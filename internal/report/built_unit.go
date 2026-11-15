package report

import (
	"iter"

	"github.com/backbone81/bar-replay-analytics/internal/events"
)

// builtUnit is a unit a player built: a builder or factory started it and it was finished in a later frame.
type builtUnit struct {
	// unitID is the unit's ID.
	unitID int

	// playerID is the ID of the player the unit was finished for.
	playerID int

	// unitDefID is the unit's unit def.
	unitDefID int

	// startFrame is the frame the unit was started at.
	startFrame int

	// finishFrame is the frame the unit was finished at.
	finishFrame int

	// builderID is the unit ID of the builder or factory which started it.
	builderID int

	// builderDefID is the unit def of the builder or factory which started it, 0 if unknown.
	builderDefID int
}

// time returns when the unit was finished.
func (u builtUnit) time() Duration {
	// The engine counts frames in an int32.
	return frameDuration(int32(u.finishFrame)) //nolint:gosec
}

// builtUnits returns the units the players with IDs below numPlayers built, in the order they were finished. Units of
// other teams, such as Gaia's, are left out, as are units no builder or factory started and units finished in the frame
// they were started at: resurrected units, which the engine creates finished once the wreck is restored
// (CBuilder::UpdateResurrect in rts/Sim/Units/UnitTypes/Builder.cpp).
func builtUnits(eventsFile *events.File, numPlayers int) iter.Seq[builtUnit] {
	return func(yield func(builtUnit) bool) {
		// Unit IDs are reused once a unit died, so each UnitCreated replaces what is known of its ID.
		unitDefIDs := make(map[int]int)
		started := make(map[int]builtUnit)
		for _, event := range eventsFile.Events {
			switch {
			case event.UnitCreated != nil:
				created := event.UnitCreated
				unitDefIDs[created.UnitID] = created.UnitDefID
				delete(started, created.UnitID)
				if created.BuilderID != nil {
					started[created.UnitID] = builtUnit{
						startFrame:   event.Frame,
						builderID:    *created.BuilderID,
						builderDefID: unitDefIDs[*created.BuilderID],
					}
				}
			case event.UnitFinished != nil:
				finished := event.UnitFinished
				unit, ok := started[finished.UnitID]
				if !ok || event.Frame == unit.startFrame || finished.UnitTeam < 0 || finished.UnitTeam >= numPlayers {
					continue
				}
				unit.unitID = finished.UnitID
				unit.playerID = finished.UnitTeam
				unit.unitDefID = finished.UnitDefID
				unit.finishFrame = event.Frame
				if !yield(unit) {
					return
				}
			}
		}
	}
}

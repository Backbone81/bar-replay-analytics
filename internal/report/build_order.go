package report

import (
	"fmt"
	"math"

	"github.com/backbone81/bar-replay-analytics-go/internal/events"
)

// BuildOrderEntry is one entry of a build order: a unit or building a player built.
type BuildOrderEntry struct {
	// Time is when the unit was finished.
	Time Duration `json:"time"`

	// Name is the name the BAR UI shows, like "Bot Lab".
	Name string `json:"name"`

	// BuildRate is how fast the unit was built, relative to the builder which started it building alone with enough
	// resources: the unit's build time divided by the builder's build speed, over the time from start to finish,
	// rounded to two decimals. 1 means there were enough resources. Below 1 means the team lacked metal or energy, the
	// builder paused, or a gadget slowed the build steps (AllowUnitBuildStep, e.g. builder priority). Above 1 means
	// other builders assisted. How much build power they added is unknown, so above 1 the rate does not tell whether
	// there were enough resources for all of them.
	BuildRate float64 `json:"buildRate"`
}

// startedUnit is a unit a builder or factory started, for the build order entry when it is finished.
type startedUnit struct {
	// frame is the frame the unit was started at.
	frame int

	// builderDefID is the unit def of the builder or factory which started it, 0 if unknown.
	builderDefID int
}

// newOpeningBuildOrders prepares the build order of the opening of each of numPlayers players, by player ID: every unit
// and building a player's builders and factories built and finished before Config.OpeningLength, in the order they were
// finished. Units finished in the frame they were started at are left out: resurrected units, which the engine
// creates finished once the wreck is restored (CBuilder::UpdateResurrect in rts/Sim/Units/UnitTypes/Builder.cpp).
// Returns an error if the builder of a unit in a build order has no build speed.
func newOpeningBuildOrders(eventsFile *events.File, numPlayers int, config Config) ([][]BuildOrderEntry, error) {
	buildOrders := make([][]BuildOrderEntry, numPlayers)
	for i := range buildOrders {
		buildOrders[i] = []BuildOrderEntry{}
	}

	openingEnd := frames(config.OpeningLength)
	// Unit IDs are reused once a unit died, so each UnitCreated replaces what is known of its ID.
	unitDefIDs := make(map[int]int)
	started := make(map[int]startedUnit)
	for _, event := range eventsFile.Events {
		if float64(event.Frame) >= openingEnd {
			break
		}
		switch {
		case event.UnitCreated != nil:
			created := event.UnitCreated
			unitDefIDs[created.UnitID] = created.UnitDefID
			delete(started, created.UnitID)
			if created.BuilderID != nil {
				started[created.UnitID] = startedUnit{
					frame:        event.Frame,
					builderDefID: unitDefIDs[*created.BuilderID],
				}
			}
		case event.UnitFinished != nil:
			finished := event.UnitFinished
			unit, ok := started[finished.UnitID]
			if !ok || event.Frame == unit.frame || finished.UnitTeam < 0 || finished.UnitTeam >= numPlayers {
				continue
			}
			builder := eventsFile.UnitDefs[unit.builderDefID]
			if builder.BuildSpeed <= 0 {
				return nil, fmt.Errorf("unit %d finished at frame %d: its builder %q has no build speed",
					finished.UnitID, event.Frame, builder.Name)
			}
			unitDef := eventsFile.UnitDefs[finished.UnitDefID]
			buildOrders[finished.UnitTeam] = append(buildOrders[finished.UnitTeam], BuildOrderEntry{
				// The engine counts frames in an int32.
				Time:      frameDuration(int32(event.Frame)), //nolint:gosec
				Name:      unitDef.TranslatedHumanName,
				BuildRate: buildRate(unitDef, builder, event.Frame-unit.frame),
			})
		}
	}
	return buildOrders, nil
}

// buildRate returns the build rate (BuildOrderEntry.BuildRate) of a unit of unitDef which builder took buildFrames to
// build. A builder adds its build speed per second to the unit's build progress, measured in build time, but only in
// frames its team can pay for (CUnit::AddBuildPower in rts/Sim/Units/Unit.cpp). builder must have a build speed and
// buildFrames must be positive.
func buildRate(unitDef events.UnitDef, builder events.UnitDef, buildFrames int) float64 {
	nominalFrames := unitDef.BuildTime / builder.BuildSpeed * GameSpeed
	return math.Round(nominalFrames/float64(buildFrames)*100) / 100
}

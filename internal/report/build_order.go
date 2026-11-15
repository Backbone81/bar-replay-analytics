package report

import (
	"fmt"

	"github.com/backbone81/bar-replay-analytics/internal/events"
)

// BuildOrderEntry is one entry of a build order: a unit or building a player built.
type BuildOrderEntry struct {
	// Time is when the unit was finished.
	Time Duration `json:"time"`

	// Name is the name the BAR UI shows, like "Bot Lab".
	Name string `json:"name"`

	// BuildRate is the unit's build time divided by the build speed of the builder which started it, over the time from
	// start to finish, rounded to two decimals. 1 is that builder alone with enough resources. Below 1, the team lacked
	// metal or energy, the builder paused, or a gadget slowed the build (AllowUnitBuildStep, e.g. builder priority).
	// Above 1, other builders assisted; whether there were enough resources for all of them is then unknown.
	BuildRate float64 `json:"buildRate"`
}

// newOpeningBuildOrders prepares the opening build order of each of numPlayers players, by player ID: every unit and
// building a player built (see builtUnits) and finished before Config.OpeningBuildOrderLength, in the order they were
// finished. Returns an error if the builder of a unit in a build order has no build speed.
func newOpeningBuildOrders(eventsFile *events.File, numPlayers int, config Config) ([][]BuildOrderEntry, error) {
	buildOrders := make([][]BuildOrderEntry, numPlayers)
	for i := range buildOrders {
		buildOrders[i] = []BuildOrderEntry{}
	}

	buildOrderEnd := frames(config.OpeningBuildOrderLength)
	for unit := range builtUnits(eventsFile, numPlayers) {
		if float64(unit.finishFrame) >= buildOrderEnd {
			break
		}
		builder := eventsFile.UnitDefs[unit.builderDefID]
		if builder.BuildSpeed <= 0 {
			return nil, fmt.Errorf("unit %d finished at frame %d: its builder %q has no build speed",
				unit.unitID, unit.finishFrame, builder.Name)
		}
		unitDef := eventsFile.UnitDefs[unit.unitDefID]
		buildOrders[unit.playerID] = append(buildOrders[unit.playerID], BuildOrderEntry{
			Time:      unit.time(),
			Name:      unitDef.TranslatedHumanName,
			BuildRate: buildRate(unitDef, builder, unit.finishFrame-unit.startFrame),
		})
	}
	return buildOrders, nil
}

// buildRate returns the build rate (BuildOrderEntry.BuildRate) of a unit of unitDef which builder took buildFrames to
// build. A builder adds its build speed per second to the unit's build progress, measured in build time, but only in
// frames its team can pay for (CUnit::AddBuildPower in rts/Sim/Units/Unit.cpp). builder must have a build speed and
// buildFrames must be positive.
func buildRate(unitDef events.UnitDef, builder events.UnitDef, buildFrames int) float64 {
	nominalFrames := unitDef.BuildTime / builder.BuildSpeed * GameSpeed
	return roundTo2(nominalFrames / float64(buildFrames))
}

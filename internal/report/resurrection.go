package report

import (
	"math"

	"github.com/backbone81/bar-replay-analytics-go/internal/events"
)

// ResurrectEnergyCostFactor is the part of a unit's energy cost a resurrection of it costs, BAR's resurrect
// energyCostFactor in gamedata/modrules.lua. A resurrection costs no metal (CBuilder::UpdateResurrect in
// rts/Sim/Units/UnitTypes/Builder.cpp, CModInfo::resurrectCostFactor in rts/Sim/Misc/ModInfo.cpp).
const ResurrectEnergyCostFactor = 0.5

// Resurrection is what a player's resurrection bots brought back from wrecks, from the units resurrected in the
// events: a unit a builder which can resurrect created, finished in the same frame, as the engine creates a unit once
// its wreck is restored (CBuilder::UpdateResurrect).
//
// Its EnergySpent is the energy spent on finished resurrections only. A resurrection which was given up also cost
// energy, and a wreck already partly reclaimed is first restored with the metal and energy reclaimed from it
// (CFeature::AddBuildPower in rts/Sim/Features/Feature.cpp); the events show neither.
type Resurrection struct {
	// Total provides what the player resurrected over the whole game.
	Total Resurrected `json:"total"`

	// Intervals provides what the player resurrected in the intervals of Economy.Intervals. Resurrections at the end of
	// an interval count in it, those after the last one in the last one.
	// Is empty for a game of no length (Metadata.Duration).
	Intervals []ResurrectionInterval `json:"intervals"`
}

// ResurrectionInterval is what a player resurrected in one interval of the game.
type ResurrectionInterval struct {
	Resurrected

	// Time is the span of the game this interval covers.
	Time Interval `json:"time"`
}

// Resurrected provides what the units a player resurrected were worth and what they cost.
type Resurrected struct {
	// MetalValue is the metal cost of the units, rounded to whole metal: what building them would have cost. The
	// resurrection itself costs no metal.
	MetalValue float64 `json:"metalValue"`

	// EnergySpent is the energy spent on resurrecting them, rounded to whole energy: ResurrectEnergyCostFactor of their
	// energy cost.
	EnergySpent float64 `json:"energySpent"`
}

// resurrectedUnit is a unit a player resurrected.
type resurrectedUnit struct {
	// frame is the frame the unit was resurrected at.
	frame int

	// metalValue is the unit's metal cost.
	metalValue float64

	// energySpent is the energy its resurrection cost.
	energySpent float64
}

// resurrectionHistory provides the units a player resurrected, oldest first.
type resurrectionHistory []resurrectedUnit

// newResurrectionHistories prepares the resurrection history of each of the numPlayers players, by player ID, from the
// units resurrected in the events. Units of other teams, such as Gaia's, are left out.
func newResurrectionHistories(eventsFile *events.File, numPlayers int) []resurrectionHistory {
	histories := make([]resurrectionHistory, numPlayers)
	// Unit IDs are reused once a unit died, so each UnitCreated replaces what is known of its ID.
	unitDefIDs := make(map[int]int)
	resurrectedAt := make(map[int]int)
	for _, event := range eventsFile.Events {
		switch {
		case event.UnitCreated != nil:
			created := event.UnitCreated
			unitDefIDs[created.UnitID] = created.UnitDefID
			delete(resurrectedAt, created.UnitID)
			if created.BuilderID == nil {
				continue
			}
			if builderDefID, ok := unitDefIDs[*created.BuilderID]; ok && eventsFile.UnitDefs[builderDefID].CanResurrect {
				resurrectedAt[created.UnitID] = event.Frame
			}
		case event.UnitFinished != nil:
			finished := event.UnitFinished
			frame, ok := resurrectedAt[finished.UnitID]
			delete(resurrectedAt, finished.UnitID)
			if !ok || frame != event.Frame || finished.UnitTeam < 0 || finished.UnitTeam >= numPlayers {
				continue
			}
			unitDef := eventsFile.UnitDefs[finished.UnitDefID]
			histories[finished.UnitTeam] = append(histories[finished.UnitTeam], resurrectedUnit{
				frame:       event.Frame,
				metalValue:  unitDef.MetalCost,
				energySpent: unitDef.EnergyCost * ResurrectEnergyCostFactor,
			})
		}
	}
	return histories
}

// newResurrection prepares the resurrection from a player's resurrection history, in the intervals of spans
// (newIntervalSpans).
func newResurrection(history resurrectionHistory, spans []frameSpan) Resurrection {
	resurrection := Resurrection{Intervals: []ResurrectionInterval{}}
	for _, unit := range history {
		resurrection.Total.add(unit)
	}
	resurrection.Total = resurrection.Total.rounded()
	next := 0
	for i, span := range spans {
		interval := ResurrectionInterval{Time: span.interval()}
		isLast := i == len(spans)-1
		for ; next < len(history) && (isLast || history[next].frame <= span.end); next++ {
			interval.add(history[next])
		}
		interval.Resurrected = interval.rounded()
		resurrection.Intervals = append(resurrection.Intervals, interval)
	}
	return resurrection
}

// add adds a resurrected unit.
func (r *Resurrected) add(unit resurrectedUnit) {
	r.MetalValue += unit.metalValue
	r.EnergySpent += unit.energySpent
}

// rounded returns r with the metal and energy rounded to whole amounts.
func (r Resurrected) rounded() Resurrected {
	r.MetalValue = math.Round(r.MetalValue)
	r.EnergySpent = math.Round(r.EnergySpent)
	return r
}

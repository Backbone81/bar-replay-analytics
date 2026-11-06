package report

import (
	"cmp"
	"slices"
)

// LossLocationInterval is where a player lost units and buildings in one interval of the game, by the cells of the
// MapGrid. The units lost are those of Combat.
//
// Losses in an enemy's start cell, killed by buildings, are an attack into a defended base; losses of buildings far
// from the player's start, an expansion lost.
type LossLocationInterval struct {
	// Time is the span of the game this interval covers.
	Time Interval `json:"time"`

	// Cells provides the cells the player lost units or buildings in, the most metal first. Cells without losses are
	// left out.
	Cells []CellLosses `json:"cells"`
}

// CellLosses is what a player lost in one cell of the map grid.
type CellLosses struct {
	// Cell is the name of the cell, like "C2" (see MapGrid).
	Cell string `json:"cell"`

	// Units provides the mobile units lost.
	Units DeadUnits `json:"units"`

	// Buildings provides the buildings lost, factories included.
	Buildings DeadUnits `json:"buildings"`

	// KilledByBuildings provides the units and buildings lost which a building killed: a static defence, or a
	// long-range weapon such as a long-range plasma cannon or a nuclear missile.
	KilledByBuildings DeadUnits `json:"killedByBuildings"`
}

// newLossLocations prepares the loss locations from a player's combat history on grid, in the intervals of spans
// (lossLocationSpans). Losses at the end of an interval count in it, those after the last one in the last one.
func newLossLocations(casualties combatHistory, grid MapGrid, spans []frameSpan) []LossLocationInterval {
	intervals := make([]LossLocationInterval, 0, len(spans))
	next := 0
	for i, span := range spans {
		cells := make(map[string]*CellLosses)
		isLast := i == len(spans)-1
		for ; next < len(casualties) && (isLast || casualties[next].frame <= span.end); next++ {
			c := casualties[next]
			if c.isKill {
				continue
			}
			name := grid.cell(c.x, c.z)
			cell, ok := cells[name]
			if !ok {
				cell = &CellLosses{Cell: name}
				cells[name] = cell
			}
			if c.isBuilding {
				cell.Buildings.add(c.metal)
			} else {
				cell.Units.add(c.metal)
			}
			if c.killerIsBuilding {
				cell.KilledByBuildings.add(c.metal)
			}
		}
		interval := LossLocationInterval{Time: span.interval(), Cells: make([]CellLosses, 0, len(cells))}
		for _, cell := range cells {
			cell.Units = cell.Units.rounded()
			cell.Buildings = cell.Buildings.rounded()
			cell.KilledByBuildings = cell.KilledByBuildings.rounded()
			interval.Cells = append(interval.Cells, *cell)
		}
		slices.SortFunc(interval.Cells, func(a, b CellLosses) int {
			return cmp.Or(
				cmp.Compare(b.Units.Metal+b.Buildings.Metal, a.Units.Metal+a.Buildings.Metal),
				cmp.Compare(b.Units.Count+b.Buildings.Count, a.Units.Count+a.Buildings.Count),
				cmp.Compare(a.Cell, b.Cell),
			)
		})
		intervals = append(intervals, interval)
	}
	return intervals
}

// lossLocationSpans returns the spans of the loss locations: spans (newIntervalSpans) with those of the opening merged
// into one, as there is little fighting in the opening, and those after the opening as they are. config must be valid
// (Config.Validate).
func lossLocationSpans(spans []frameSpan, config Config) []frameSpan {
	opening := frames(config.OpeningLength)
	var result []frameSpan
	for _, span := range spans {
		if len(result) == 1 && float64(span.start) < opening {
			result[0].end = span.end
			continue
		}
		result = append(result, span)
	}
	return result
}

package report

import (
	"github.com/backbone81/bar-replay-analytics-go/internal/tdf"
)

// GameSpeed is the number of game frames per second of game time.
// See GAME_SPEED in rts/Sim/Misc/GlobalConstants.h.
const GameSpeed = 30

// withoutEmptyValues returns a copy of section without its empty values, in its subsections too. section is left
// unchanged. The copy is held by value, so that a missing section is written as an empty object.
func withoutEmptyValues(section *tdf.Section) tdf.Section {
	filtered := section.FilterValues(func(_ string, value tdf.Value) bool { return value != "" })
	if filtered == nil {
		return tdf.Section{}
	}
	return *filtered
}

package report

import (
	"slices"

	"github.com/backbone81/bar-replay-analytics-go/internal/demo"
)

// Economy is what a player produced and used of metal and energy, from its TeamStatistics history.
type Economy struct {
	// Total provides the amounts over the whole game.
	Total Resources `json:"total"`

	// PerSecond provides the average amounts per second over the whole game.
	PerSecond Resources `json:"perSecond"`

	// Intervals provides the average amounts per second in consecutive intervals of Config.IntervalLength, oldest first.
	// The last interval ends with the game, so it may be shorter.
	Intervals []EconomyInterval `json:"intervals"`
}

// EconomyInterval is the economy in one interval of the game.
type EconomyInterval struct {
	// Time is the span of the game this interval covers.
	Time Interval `json:"time"`

	// PerSecond provides the average amounts per second in this interval.
	PerSecond Resources `json:"perSecond"`
}

// Resources provides amounts of metal and energy.
type Resources struct {
	// Metal provides the amounts of metal.
	Metal Resource `json:"metal"`

	// Energy provides the amounts of energy.
	Energy Resource `json:"energy"`
}

// Resource provides amounts of one resource, see demo.TeamStatistics.
type Resource struct {
	// Produced is the income.
	Produced float64 `json:"produced"`

	// Used is what was spent.
	Used float64 `json:"used"`

	// Wasted is what was lost because the storage was full, after sharing to allies. The engine calls it excess.
	Wasted float64 `json:"wasted"`

	// Received is what allies gave or shared.
	Received float64 `json:"received"`

	// Sent is what was given or shared to allies.
	Sent float64 `json:"sent"`
}

// newEconomy prepares the economy from a player's statistics history. endFrame is the frame the game ended at,
// intervalFrames the length of an interval in frames.
// Returns nil if the history is empty.
func newEconomy(history []demo.TeamStatistics, endFrame int64, intervalFrames float64) *Economy {
	if len(history) == 0 {
		return nil
	}
	history = endHistory(history, endFrame)

	last := history[len(history)-1]
	economy := &Economy{
		Total:     resources(last),
		Intervals: []EconomyInterval{},
	}
	if last.Frame > 0 {
		economy.PerSecond = economy.Total.div(seconds(last.Frame))
	}
	bounds := intervalBounds(history, intervalFrames)
	for i := 1; i < len(bounds); i++ {
		first, current := history[bounds[i-1]], history[bounds[i]]
		economy.Intervals = append(economy.Intervals, EconomyInterval{
			Time:      frameInterval(first.Frame, current.Frame),
			PerSecond: resources(current).sub(resources(first)).div(seconds(current.Frame - first.Frame)),
		})
	}
	return economy
}

// intervalBounds returns the indexes of the history entries which start and end the intervals: an interval ends at
// the first entry at least intervalFrames after its start, the last interval at the last entry. Intervals are thus as
// long as intervalFrames, rounded up to whole statistics periods, except for the last one, which may be shorter.
func intervalBounds(history []demo.TeamStatistics, intervalFrames float64) []int {
	bounds := []int{0}
	for i := 1; i < len(history); i++ {
		frames := float64(history[i].Frame - history[bounds[len(bounds)-1]].Frame)
		if frames >= intervalFrames || (i == len(history)-1 && frames > 0) {
			bounds = append(bounds, i)
		}
	}
	return bounds
}

// endHistory returns a copy of history whose last entry, the totals at the end of the game, is at endFrame.
//
// The engine labels the totals in advance with the frame of the next entry, up to TeamStatPeriod seconds after the end
// of the game (demo.TeamStatistics.Frame). If the totals did not change since the entry before, they are dropped. If
// the game ended at that entry's frame (endFrame is in whole seconds, so it may lie slightly before the actual end),
// the totals replace that entry, which keeps them.
func endHistory(history []demo.TeamStatistics, endFrame int64) []demo.TeamStatistics {
	ended := slices.Clone(history)
	count := len(ended)
	if count >= 2 && sameTotals(ended[count-1], ended[count-2]) {
		return ended[:count-1]
	}

	last := &ended[count-1]
	// The minimum is at most last.Frame, so it fits an int32.
	last.Frame = int32(min(int64(last.Frame), endFrame)) //nolint:gosec
	if count >= 2 && last.Frame <= ended[count-2].Frame {
		last.Frame = ended[count-2].Frame
		ended[count-2] = *last
		return ended[:count-1]
	}
	return ended
}

// sameTotals returns whether a and b hold the same totals, regardless of their frames.
func sameTotals(a, b demo.TeamStatistics) bool {
	a.Frame = b.Frame
	return a == b
}

// resources returns the resource totals of a statistics entry.
func resources(stats demo.TeamStatistics) Resources {
	return Resources{
		Metal: Resource{
			Produced: float64(stats.MetalProduced),
			Used:     float64(stats.MetalUsed),
			Wasted:   float64(stats.MetalExcess),
			Received: float64(stats.MetalReceived),
			Sent:     float64(stats.MetalSent),
		},
		Energy: Resource{
			Produced: float64(stats.EnergyProduced),
			Used:     float64(stats.EnergyUsed),
			Wasted:   float64(stats.EnergyExcess),
			Received: float64(stats.EnergyReceived),
			Sent:     float64(stats.EnergySent),
		},
	}
}

// sub returns r minus other.
func (r Resources) sub(other Resources) Resources {
	return Resources{
		Metal:  r.Metal.sub(other.Metal),
		Energy: r.Energy.sub(other.Energy),
	}
}

// div returns r divided by divisor.
func (r Resources) div(divisor float64) Resources {
	return Resources{
		Metal:  r.Metal.div(divisor),
		Energy: r.Energy.div(divisor),
	}
}

// sub returns r minus other.
func (r Resource) sub(other Resource) Resource {
	return Resource{
		Produced: r.Produced - other.Produced,
		Used:     r.Used - other.Used,
		Wasted:   r.Wasted - other.Wasted,
		Received: r.Received - other.Received,
		Sent:     r.Sent - other.Sent,
	}
}

// div returns r divided by divisor.
func (r Resource) div(divisor float64) Resource {
	return Resource{
		Produced: r.Produced / divisor,
		Used:     r.Used / divisor,
		Wasted:   r.Wasted / divisor,
		Received: r.Received / divisor,
		Sent:     r.Sent / divisor,
	}
}

// seconds returns a frame as seconds of game time.
func seconds(frame int32) float64 {
	return float64(frame) / GameSpeed
}

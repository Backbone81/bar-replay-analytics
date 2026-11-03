package report

import (
	"time"

	"github.com/backbone81/bar-replay-analytics-go/internal/demo"
	"github.com/backbone81/bar-replay-analytics-go/internal/events"
)

// Economy is what a player produced and used of metal and energy, from its TeamStatistics history (economyHistory),
// and the metal extractors it held, from the events.
type Economy struct {
	// Total provides the amounts over the whole game.
	Total Resources `json:"total"`

	// PerSecond provides the average amounts per second over the whole game.
	PerSecond Resources `json:"perSecond"`

	// Intervals provides the average amounts per second in consecutive intervals, oldest first: of
	// Config.OpeningIntervalLength in the opening, of Config.IntervalLength after it. The last interval ends with the
	// game, so it may be shorter.
	Intervals []EconomyInterval `json:"intervals"`
}

// EconomyInterval is the economy in one interval of the game.
type EconomyInterval struct {
	// Time is the span of the game this interval covers.
	Time Interval `json:"time"`

	// PerSecond provides the average amounts per second in this interval.
	PerSecond Resources `json:"perSecond"`

	// MetalExtractors provides the metal extractors the player held at the end of this interval.
	MetalExtractors MetalExtractors `json:"metalExtractors"`
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

// economyHistory returns the statistics history of the player with playerID: the replay's, unless the events file has
// more entries for it. A replay saved by the game itself holds an entry every 15 s, but one downloaded from the replay
// database holds only the totals at the end, and a replay of a crashed game none. The playback records the history
// anew (events.TeamStats), so the events file has every entry. Their last entry is labeled with the frame the widget
// read them at, not with that of the next period, which entryFrame handles alike.
// Returns nil if neither has statistics of the player.
func economyHistory(replay *demo.Demo, eventsFile *events.File, playerID int) []demo.TeamStatistics {
	var history []demo.TeamStatistics
	if playerID < len(replay.TeamStats) {
		history = replay.TeamStats[playerID]
	}
	eventsHistory := eventsFile.TeamStats[playerID]
	if len(eventsHistory) <= len(history) {
		return history
	}
	history = make([]demo.TeamStatistics, 0, len(eventsHistory))
	for _, stats := range eventsHistory {
		history = append(history, teamStatistics(stats))
	}
	return history
}

// teamStatistics returns an entry of the events file's statistics history as the replay's TeamStatistics, whose
// fields the engine holds as float32 and int32.
func teamStatistics(stats events.TeamStats) demo.TeamStatistics {
	//nolint:gosec // The engine's TeamStatistics are int32, so the counts fit.
	return demo.TeamStatistics{
		Frame:            int32(stats.Frame),
		MetalUsed:        float32(stats.MetalUsed),
		EnergyUsed:       float32(stats.EnergyUsed),
		MetalProduced:    float32(stats.MetalProduced),
		EnergyProduced:   float32(stats.EnergyProduced),
		MetalExcess:      float32(stats.MetalExcess),
		EnergyExcess:     float32(stats.EnergyExcess),
		MetalReceived:    float32(stats.MetalReceived),
		EnergyReceived:   float32(stats.EnergyReceived),
		MetalSent:        float32(stats.MetalSent),
		EnergySent:       float32(stats.EnergySent),
		DamageDealt:      float32(stats.DamageDealt),
		DamageReceived:   float32(stats.DamageReceived),
		UnitsProduced:    int32(stats.UnitsProduced),
		UnitsDied:        int32(stats.UnitsDied),
		UnitsReceived:    int32(stats.UnitsReceived),
		UnitsSent:        int32(stats.UnitsSent),
		UnitsCaptured:    int32(stats.UnitsCaptured),
		UnitsOutCaptured: int32(stats.UnitsOutCaptured),
		UnitsKilled:      int32(stats.UnitsKilled),
	}
}

// newEconomy prepares the economy from a player's statistics history and metal extractor history. endFrame is the
// frame the game ended at, config sets the intervals and must be valid (Config.Validate).
// Returns nil if the statistics history is empty.
func newEconomy(
	history []demo.TeamStatistics,
	metalExtractors metalExtractorHistory,
	endFrame int64,
	config Config,
) *Economy {
	if len(history) == 0 {
		return nil
	}

	economy := &Economy{
		Total:     resources(history[len(history)-1]),
		Intervals: []EconomyInterval{},
	}
	if lastFrame := entryFrame(history, len(history)-1, endFrame); lastFrame > 0 {
		economy.PerSecond = economy.Total.div(seconds(lastFrame))
	}
	bounds := intervalBounds(history, endFrame, config)
	for i := 1; i < len(bounds); i++ {
		first, current := bounds[i-1], bounds[i]
		start, end := entryFrame(history, first, endFrame), entryFrame(history, current, endFrame)
		economy.Intervals = append(economy.Intervals, EconomyInterval{
			Time:            frameInterval(start, end),
			PerSecond:       resources(history[current]).sub(resources(history[first])).div(seconds(end - start)),
			MetalExtractors: metalExtractors.at(end),
		})
	}
	return economy
}

// entryFrame returns the frame of history entry i. The last entry holds the totals at the end of the game, but the
// engine labels it with the frame of the next period (demo.TeamStatistics.Frame), so its frame is capped at endFrame,
// though not earlier than the entry before it: endFrame is in whole seconds, so it may lie slightly before the actual
// end.
func entryFrame(history []demo.TeamStatistics, i int, endFrame int64) int32 {
	frame := history[i].Frame
	if i < len(history)-1 {
		return frame
	}
	// The minimum is at most frame, so it fits an int32.
	frame = int32(min(int64(frame), endFrame)) //nolint:gosec
	if i > 0 {
		frame = max(frame, history[i-1].Frame)
	}
	return frame
}

// intervalEnd returns the earliest frame at which the interval starting at the frame start may end: one opening
// interval later, but not past the end of the opening, if it starts in the opening, else one interval later.
func intervalEnd(config Config, start float64) float64 {
	opening := frames(config.OpeningLength)
	if start < opening {
		return min(start+frames(config.OpeningIntervalLength), opening)
	}
	return start + frames(config.IntervalLength)
}

// intervalBounds returns the indexes of the history entries which start and end the intervals of config: an interval
// ends at the first entry at or after its end (intervalEnd), the last interval at the last entry. Intervals are thus
// as long as config says, rounded up to whole statistics periods, except for the last one, which may be shorter. The
// frames of the entries are those of entryFrame. If the last entry has the frame of the last bound, it replaces that
// bound, as it holds the totals at the end of the game.
func intervalBounds(history []demo.TeamStatistics, endFrame int64, config Config) []int {
	bounds := []int{0}
	last := len(history) - 1
	for i := 1; i <= last; i++ {
		start := entryFrame(history, bounds[len(bounds)-1], endFrame)
		frame := entryFrame(history, i, endFrame)
		switch {
		case float64(frame) >= intervalEnd(config, float64(start)):
			bounds = append(bounds, i)
		case i == last && frame > start:
			bounds = append(bounds, i)
		case i == last:
			bounds[len(bounds)-1] = i
		}
	}
	return bounds
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

// frames returns a duration of game time in frames.
func frames(duration time.Duration) float64 {
	return duration.Seconds() * GameSpeed
}

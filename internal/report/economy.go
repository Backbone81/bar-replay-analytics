package report

import (
	"sort"
	"time"

	"github.com/backbone81/bar-replay-analytics/internal/demo"
	"github.com/backbone81/bar-replay-analytics/internal/events"
)

// Economy is what a player produced and used of metal and energy, from its TeamStatistics history (economyHistory),
// and per interval its metal extractors and builders, from the events.
type Economy struct {
	// Total provides the amounts over the whole game.
	Total Resources `json:"total"`

	// PerSecond provides the average amounts per second over the whole game.
	PerSecond Resources `json:"perSecond"`

	// Intervals provides the economy in the intervals of Config, oldest first. The last one ends with the game, so it
	// may be shorter. Is empty if the statistics history holds only the totals.
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

	// Builders provides how many of the player's commanders and constructors (see Builder) were alive and idle in this
	// interval, by class. Idle builders while resources pile up are wasted build power; while all income is spent,
	// they are not.
	Builders BuilderClasses `json:"builders"`
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
// more entries for it. A replay saved by the game holds an entry every 15 s, one downloaded from the replay database
// only the totals, one of a crashed game none; the playback records the whole history (events.TeamStats).
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

// newEconomy prepares the economy from a player's statistics history, metal extractor history and builder activity, in
// the intervals of spans (newIntervalSpans) of a game which ended at endFrame. The amounts of an interval are the
// difference of the entries at its start and end (statsAt), so it has none if the history holds only the totals.
// Returns nil if the statistics history is empty.
func newEconomy(
	history []demo.TeamStatistics,
	metalExtractors metalExtractorHistory,
	builders builderActivity,
	spans []frameSpan,
	endFrame int,
) *Economy {
	if len(history) == 0 {
		return nil
	}

	economy := &Economy{
		Total:     resources(history[len(history)-1]),
		Intervals: []EconomyInterval{},
	}
	if endFrame > 0 {
		economy.PerSecond = economy.Total.div(seconds(endFrame))
	}
	if len(history) < 2 {
		return economy
	}
	for i, span := range spans {
		start := statsAt(history, span.start, false)
		end := statsAt(history, span.end, i == len(spans)-1)
		economy.Intervals = append(economy.Intervals, EconomyInterval{
			Time:            span.interval(),
			PerSecond:       resources(end).sub(resources(start)).div(seconds(span.end - span.start)),
			MetalExtractors: metalExtractors.at(span.end),
			Builders:        builders.use(span),
		})
	}
	return economy
}

// statsAt returns the totals of history at frame: those of the last entry recorded at or before it, none before the
// first entry. If isEnd, frame is the end of the game, and the last entry holds the totals: the engine labels it with
// the frame of the next period (demo.TeamStatistics.Frame), the playback with the frame it read them at, which may be
// slightly after the end, as the game's length is in whole seconds.
func statsAt(history []demo.TeamStatistics, frame int, isEnd bool) demo.TeamStatistics {
	if isEnd {
		return history[len(history)-1]
	}
	i := sort.Search(len(history), func(i int) bool { return int(history[i].Frame) > frame })
	if i == 0 {
		return demo.TeamStatistics{}
	}
	return history[i-1]
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
func seconds(frame int) float64 {
	return float64(frame) / GameSpeed
}

// frames returns a duration of game time in frames.
func frames(duration time.Duration) float64 {
	return duration.Seconds() * GameSpeed
}

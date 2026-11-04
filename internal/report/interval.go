package report

import (
	"encoding/json"
	"math"
)

// Interval is a span of the game from Start to End. JSON writes it as start-end in minutes:seconds, like
// "15:00-20:00", the way players give a span of the game.
type Interval struct {
	// Start is when this interval starts.
	Start Duration

	// End is when this interval ends.
	End Duration
}

// frameInterval returns the interval from the frame start to the frame end.
func frameInterval(start int32, end int32) Interval {
	return Interval{
		Start: frameDuration(start),
		End:   frameDuration(end),
	}
}

// String returns the interval as start-end, see Duration.String, like "0:00-5:00" or "60:00-60:33".
func (i Interval) String() string {
	return i.Start.String() + "-" + i.End.String()
}

// MarshalJSON writes the interval as a string, see String.
func (i Interval) MarshalJSON() ([]byte, error) {
	return json.Marshal(i.String())
}

// frameSpan is a span of frames, from start to end.
type frameSpan struct {
	// start is the first frame of the span.
	start int

	// end is the frame the span ends at.
	end int
}

// newIntervalSpans returns the spans of the intervals of config in a game which ended at endFrame, with statistics
// recorded every statPeriod seconds, oldest first: an interval ends at intervalEnd, rounded up to a whole number of
// statistics periods since the start, so that it ends at a statistics entry, and the last one ends with the game, so
// it may be shorter. config must be valid (Config.Validate). Returns no spans for a game of no length.
func newIntervalSpans(endFrame int, statPeriod int, config Config) []frameSpan {
	period := float64(max(1, statPeriod*GameSpeed))
	var spans []frameSpan
	for start := 0; start < endFrame; {
		end := min(int(math.Ceil(intervalEnd(config, float64(start))/period)*period), endFrame)
		spans = append(spans, frameSpan{start: start, end: end})
		start = end
	}
	return spans
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

// interval returns the span as an Interval.
func (s frameSpan) interval() Interval {
	//nolint:gosec // The engine counts frames in an int32.
	return frameInterval(int32(s.start), int32(s.end))
}

// overlap returns the number of frames s and other have in common.
func (s frameSpan) overlap(other frameSpan) int {
	return max(0, min(s.end, other.end)-max(s.start, other.start))
}

package report

import "encoding/json"

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

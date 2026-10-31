package report

import (
	"encoding/json"
	"fmt"
	"time"
)

// Duration is a span of game time, or a point in the game as the span since its start. JSON writes it as
// minutes:seconds, like "60:33", the way players give timings.
type Duration time.Duration

// frameDuration returns the game time of frame.
func frameDuration(frame int32) Duration {
	return Duration(time.Duration(frame) * time.Second / GameSpeed)
}

// String returns the duration as minutes:seconds, rounded to whole seconds, like "0:05" or "60:33". Minutes are not
// carried over into hours.
func (d Duration) String() string {
	total := int64(time.Duration(d).Round(time.Second) / time.Second)
	sign := ""
	if total < 0 {
		sign = "-"
		total = -total
	}
	return fmt.Sprintf("%s%d:%02d", sign, total/60, total%60)
}

// MarshalJSON writes the duration as a string, see String. Go would write a time.Duration as a number of nanoseconds.
func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

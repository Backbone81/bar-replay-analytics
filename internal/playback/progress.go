package playback

import (
	"time"

	"github.com/backbone81/bar-replay-analytics-go/internal/events"
)

// Progress is how far a playback has come, see Config.OnProgress.
type Progress struct {
	// Replay is the replay played back.
	Replay *Replay

	// Elapsed is the wall clock time since the engine was started.
	Elapsed time.Duration

	// Loading is true while the engine loads: the analytics widget has not created its events file yet.
	Loading bool

	// Frame is the game frame of the last event the analytics widget wrote.
	Frame int
}

// GameTime returns the game time of Frame.
func (p Progress) GameTime() time.Duration {
	return time.Duration(p.Frame) * time.Second / events.GameSpeed
}

// Percent returns how much of the game has been played back, from 0 to 100, and whether it is known: the replay's
// header may lack the length of the game (see Replay.GameTime).
func (p Progress) Percent() (float64, bool) {
	if p.Replay.GameTime <= 0 {
		return 0, false
	}
	return min(100, 100*float64(p.GameTime())/float64(p.Replay.GameTime)), true
}

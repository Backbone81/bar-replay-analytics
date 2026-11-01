package events

// EndReason is why the widget stopped logging.
type EndReason string

const (
	// EndReasonGameOver means the game ended: the file holds the whole game.
	EndReasonGameOver EndReason = "gameover"

	// EndReasonStalled means the replay stopped advancing before the game ended.
	EndReasonStalled EndReason = "stalled"

	// EndReasonNeverStarted means the game did not start.
	EndReasonNeverStarted EndReason = "never started"

	// EndReasonShutdown means the widget was shut down before the game ended.
	EndReasonShutdown EndReason = "shutdown"
)

// End is when and why the widget stopped logging.
type End struct {
	// Frame is the game frame the widget stopped logging at.
	Frame int `json:"frame"`

	// Reason is why the widget stopped logging.
	Reason EndReason `json:"reason"`
}

package report

import (
	"github.com/backbone81/bar-replay-analytics-go/internal/demo"
)

// Report is the prepared analytics of a replay.
type Report struct {
	// Metadata describes the game.
	Metadata Metadata `json:"metadata"`

	// Players provides every player but Gaia, indexed by player ID.
	// Is empty if the game was left before game over: the engine records the statistics only then.
	Players []Player `json:"players"`
}

// NewReport prepares the report of a replay, configured by DefaultConfig changed by options.
// Returns an error if the configuration is invalid.
func NewReport(replay *demo.Demo, options ...Option) (*Report, error) {
	config := DefaultConfig
	for _, option := range options {
		option(&config)
	}
	if err := config.Validate(); err != nil {
		return nil, err
	}

	endFrame := int64(replay.FileHeader.GameTime) * GameSpeed
	players := make([]Player, len(replay.TeamStats))
	for i, history := range replay.TeamStats {
		players[i] = Player{
			PlayerID: i,
			Economy:  newEconomy(history, endFrame, config),
		}
	}

	return &Report{
		Metadata: newMetadata(replay),
		Players:  players,
	}, nil
}

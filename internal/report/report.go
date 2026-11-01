package report

import (
	"fmt"
	"slices"

	"github.com/backbone81/bar-replay-analytics-go/internal/demo"
	"github.com/backbone81/bar-replay-analytics-go/internal/tdf"
)

// Report is the prepared analytics of a replay.
type Report struct {
	// Metadata describes the game.
	Metadata Metadata `json:"metadata"`

	// Teams provides every team but Gaia's, indexed by team ID.
	Teams []Team `json:"teams"`

	// Players provides every player but Gaia, indexed by player ID.
	Players []Player `json:"players"`
}

// NewReport prepares the report of a replay from the replay and its parsed setup script, configured by DefaultConfig
// changed by options.
// Returns an error if the configuration is invalid, if the setup script is one the engine refuses or holds an invalid
// value the report reads, or if it disagrees with the statistics on the number of players.
func NewReport(replay *demo.Demo, setupScript *tdf.Section, options ...Option) (*Report, error) {
	config := DefaultConfig
	for _, option := range options {
		option(&config)
	}
	if err := config.Validate(); err != nil {
		return nil, err
	}

	game := setupScript.Section("game")
	metadata := newMetadata(replay, setupScript)
	teams, teamIDs := newTeams(game, metadata.WinningTeams)
	players, err := newPlayers(game, teamIDs, replay, config)
	if err != nil {
		return nil, fmt.Errorf("setup script: %w", err)
	}
	// Some replays hold empty histories for fewer teams than there are, which is as good as no statistics.
	hasStats := slices.ContainsFunc(replay.TeamStats, func(history []demo.TeamStatistics) bool {
		return len(history) > 0
	})
	if hasStats && len(replay.TeamStats) != len(players) {
		return nil, fmt.Errorf(
			"setup script: got %d teams, want the %d of the statistics",
			len(players),
			len(replay.TeamStats),
		)
	}
	for _, player := range players {
		teams[player.TeamID].PlayerIDs = append(teams[player.TeamID].PlayerIDs, player.PlayerID)
	}

	return &Report{
		Metadata: metadata,
		Teams:    teams,
		Players:  players,
	}, nil
}

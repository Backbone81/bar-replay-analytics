package report

import (
	"fmt"
	"slices"

	"github.com/backbone81/bar-replay-analytics-go/internal/demo"
	"github.com/backbone81/bar-replay-analytics-go/internal/events"
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

// NewReport prepares the report of a replay from the replay, its parsed setup script and the events file of its
// playback, configured by DefaultConfig changed by options.
// Returns an error if the configuration is invalid, if the setup script is one the engine refuses or holds an invalid
// value the report reads, if it disagrees with the statistics on the number of players, or if the events file is
// invalid or not of this game.
//
//nolint:funlen // One part of the report after the other reads best in one function.
func NewReport(
	replay *demo.Demo,
	setupScript *tdf.Section,
	eventsFile *events.File,
	options ...Option,
) (*Report, error) {
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
	players, err := newPlayers(game, teamIDs)
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
	if err := eventsFile.Validate(); err != nil {
		return nil, err
	}
	if err := validateEventsGame(eventsFile, metadata, len(players)); err != nil {
		return nil, fmt.Errorf("events: %w", err)
	}
	buildOrders, err := newOpeningBuildOrders(eventsFile, len(players), config)
	if err != nil {
		return nil, fmt.Errorf("events: %w", err)
	}
	firstFinished := newFirstFinished(eventsFile, len(players))
	metalExtractors, err := newMetalExtractorHistories(eventsFile, len(players))
	if err != nil {
		return nil, fmt.Errorf("events: %w", err)
	}
	endFrame := int64(replay.FileHeader.GameTime) * GameSpeed
	for i := range players {
		players[i].setFactionFromStartUnit(eventsFile)
		players[i].Economy = newEconomy(economyHistory(replay, eventsFile, i), metalExtractors[i], endFrame, config)
		players[i].OpeningBuildOrder = buildOrders[i]
		players[i].FirstFinished = firstFinished[i]
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

// validateEventsGame returns an error if the events file is not of the game of metadata with numPlayers players: if its
// map, game version or number of teams differ.
func validateEventsGame(eventsFile *events.File, metadata Metadata, numPlayers int) error {
	game := eventsFile.Game
	if game.MapName != metadata.Map {
		return fmt.Errorf("map: got %q, want %q", game.MapName, metadata.Map)
	}
	if gameVersion := game.GameName + " " + game.GameVersion; gameVersion != metadata.GameVersion {
		return fmt.Errorf("game version: got %q, want %q", gameVersion, metadata.GameVersion)
	}
	if len(eventsFile.Teams) != numPlayers {
		return fmt.Errorf("got %d teams, want the %d of the setup script", len(eventsFile.Teams), numPlayers)
	}
	return nil
}

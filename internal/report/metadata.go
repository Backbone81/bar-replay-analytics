package report

import (
	"slices"
	"time"

	"github.com/backbone81/bar-replay-analytics-go/internal/demo"
)

// Metadata describes the game of a replay.
type Metadata struct {
	// EngineVersion is the engine version, a RecoilEngine tag like "2026.07.04".
	EngineVersion string `json:"engineVersion"`

	// GameID is the unique game identifier as a lowercase hex string, which is how bar-db names the replay.
	GameID string `json:"gameID"` //nolint:tagliatelle

	// StartTime is when the game was launched, in UTC: when the engine started recording the replay, before the map
	// was loaded and the game began.
	StartTime time.Time `json:"startTime"`

	// Duration is the length of the game in game time, rounded down to whole seconds. Is 0 if the game was left before
	// game over: the engine records it only then.
	Duration Duration `json:"duration"`

	// WinningTeams provides the IDs of the teams which have won, without duplicates, sorted. Is empty if nobody won,
	// e.g. when the game was left before it ended.
	WinningTeams []int `json:"winningTeams"`
}

// newMetadata collects the metadata from the header and the winning allyteams.
func newMetadata(replay *demo.Demo) Metadata {
	winningTeams := make([]int, 0, len(replay.WinningAllyTeams))
	for _, allyTeam := range replay.WinningAllyTeams {
		if !slices.Contains(winningTeams, int(allyTeam)) {
			winningTeams = append(winningTeams, int(allyTeam))
		}
	}
	slices.Sort(winningTeams)

	header := replay.FileHeader
	return Metadata{
		EngineVersion: header.VersionStringAsString(),
		GameID:        header.GameIDAsString(),
		// The demo package validated that UnixTime fits an int64.
		StartTime:    time.Unix(int64(header.UnixTime), 0).UTC(), //nolint:gosec
		Duration:     Duration(time.Duration(header.GameTime) * time.Second),
		WinningTeams: winningTeams,
	}
}

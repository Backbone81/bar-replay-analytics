package report

import (
	"slices"
	"time"

	"github.com/backbone81/bar-replay-analytics-go/internal/demo"
	"github.com/backbone81/bar-replay-analytics-go/internal/tdf"
)

// Metadata describes the game of a replay.
type Metadata struct {
	// EngineVersion is the engine version, a RecoilEngine tag like "2026.07.04".
	EngineVersion string `json:"engineVersion"`

	// GameVersion is the game version, like "Beyond All Reason test-31756-0d4c3c3": the rapid package of BAR commit
	// 0d4c3c3. The setup script calls it gametype, bar-db gameVersion.
	GameVersion string `json:"gameVersion"`

	// Map is the name of the map, like "Argent Strata 1.1".
	Map string `json:"map"`

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

	// ModOptions provides the game options of the setup script (its modoptions section), without empty values, as
	// Spring.GetModOptions returns them to the game. Their meanings and defaults are in BAR's modoptions.lua. A game
	// started from the local lobby holds only the options changed from their defaults (and some the lobby always
	// sets, such as date_day), a hosted game holds every option.
	ModOptions tdf.Section `json:"modOptions"`

	// MapOptions provides the map options of the setup script (its mapoptions section), without empty values. Their
	// meanings and defaults are in the map's mapoptions.lua.
	MapOptions tdf.Section `json:"mapOptions"`
}

// newMetadata collects the metadata from the header, the winning allyteams and the setup script.
func newMetadata(replay *demo.Demo, script *tdf.Section) Metadata {
	game := script.Section("game")
	header := replay.FileHeader
	mapName, _ := game.Value("mapname")
	gameType, _ := game.Value("gametype")
	return Metadata{
		EngineVersion: header.VersionStringAsString(),
		GameVersion:   gameType.String(),
		Map:           mapName.String(),
		GameID:        header.GameIDAsString(),
		// The demo package validated that UnixTime fits an int64.
		StartTime:    time.Unix(int64(header.UnixTime), 0).UTC(), //nolint:gosec
		Duration:     Duration(time.Duration(header.GameTime) * time.Second),
		WinningTeams: winningTeams(replay.WinningAllyTeams),
		ModOptions:   withoutEmptyValues(game.Section("modoptions")),
		MapOptions:   withoutEmptyValues(game.Section("mapoptions")),
	}
}

// winningTeams returns the winning allyteams as team IDs, without duplicates, sorted.
func winningTeams(allyTeams []uint8) []int {
	result := make([]int, 0, len(allyTeams))
	for _, allyTeam := range allyTeams {
		if !slices.Contains(result, int(allyTeam)) {
			result = append(result, int(allyTeam))
		}
	}
	slices.Sort(result)
	return result
}

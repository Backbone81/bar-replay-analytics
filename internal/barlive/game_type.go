package barlive

// GameType is the game type of a leaderboard, as Teiserver names it in LeaderboardSeason.GameTypes and
// Leaderboard.Name.
//
// Teiserver decides the game type of a match from its number of allyteams and the size of its largest allyteam, see
// game_type in lib/teiserver/battle/libs/match_lib.ex in beyond-all-reason/teiserver.
type GameType string

// The game types with a leaderboard.
const (
	GameTypeDuel      GameType = "Duel"
	GameTypeSmallTeam GameType = "Small Team"
	GameTypeLargeTeam GameType = "Large Team"
	GameTypeFFA       GameType = "FFA"
)

// SmallTeamMaxSize is the largest allyteam of a Small Team game; a larger one makes it a Large Team game. It is the
// default of Teiserver's site config "lobby.Small team game limit" (lib/teiserver/libs/teiserver_configs.ex), which the
// public API does not expose.
const SmallTeamMaxSize = 5

// ReplayPreset returns the bar-db preset of the replays of gameType, or false if the package cannot select replays of
// gameType.
func (t GameType) ReplayPreset() (ReplayPreset, bool) {
	switch t {
	case GameTypeDuel:
		return ReplayPresetDuel, true
	case GameTypeSmallTeam, GameTypeLargeTeam:
		return ReplayPresetTeam, true
	case GameTypeFFA:
		return ReplayPresetFFA, true
	default:
		return "", false
	}
}

// Matches reports whether the game of replay has the shape of gameType: the number of allyteams and players Teiserver
// decides the game type from, and no AI. It counts players instead of teams, so a team shared by two players counts
// twice. Allyteams without players or AIs are not counted. The preset of the replay is not checked, as a ReplaySummary
// does not hold it; ReplayPreset selects it in the search.
func (t GameType) Matches(replay *ReplaySummary) bool {
	allyTeamCount := 0
	maxSize := 0
	for _, allyTeam := range replay.AllyTeams {
		if len(allyTeam.AIs) != 0 {
			return false
		}
		size := len(allyTeam.Players)
		if size == 0 {
			continue
		}
		maxSize = max(maxSize, size)
		allyTeamCount++
	}

	switch t {
	case GameTypeDuel:
		return allyTeamCount == 2 && maxSize == 1
	case GameTypeSmallTeam:
		return allyTeamCount == 2 && maxSize > 1 && maxSize <= SmallTeamMaxSize
	case GameTypeLargeTeam:
		return allyTeamCount == 2 && maxSize > SmallTeamMaxSize
	case GameTypeFFA:
		return allyTeamCount > 2 && maxSize == 1
	default:
		return false
	}
}

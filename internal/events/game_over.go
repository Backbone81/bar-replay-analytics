package events

// GameOver is written when the game ends.
type GameOver struct {
	// WinningAllyTeamIDs provides the IDs of the allyteams which have won.
	WinningAllyTeamIDs []int `json:"winningAllyTeamIDs"`
}

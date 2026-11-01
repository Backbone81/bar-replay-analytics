package events

// GameOver is written by widget:GameOver: the game ends.
type GameOver struct {
	// WinningAllyTeams provides the IDs of the allyteams which have won. Is empty if the game was undecided.
	WinningAllyTeams []int `json:"winningAllyTeams"`
}

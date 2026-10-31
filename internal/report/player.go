package report

// Player is one player of the game, human or AI: the units and resources one player or AI controls, or several players
// sharing control. The engine calls this a team.
type Player struct {
	// PlayerID is the ID of this player: the engine's team ID.
	PlayerID int `json:"playerID"` //nolint:tagliatelle

	// Economy provides the resources this player produced and used.
	// Is nil if the player has no statistics.
	Economy *Economy `json:"economy"`
}

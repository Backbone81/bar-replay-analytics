package events

// TeamDied is written by widget:TeamDied: a team died, because its players resigned or left, or it was wiped out. BAR
// then removes the team's units (game_team_death_effect.lua): every UnitDestroyed of the team from this frame on is
// such a removal, not a loss in a fight.
type TeamDied struct {
	// TeamID is the team's ID.
	TeamID int `json:"teamID"` //nolint:tagliatelle // the engine's name
}

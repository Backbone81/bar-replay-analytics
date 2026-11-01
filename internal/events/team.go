package events

// Team is a team other than Gaia, from Spring.GetTeamInfo at the end of the game.
type Team struct {
	// TeamID is the team's ID.
	TeamID int `json:"teamID"` //nolint:tagliatelle // the engine's name

	// AllyTeam is the ID of the team's allyteam.
	AllyTeam int `json:"allyTeam"`

	// IsAI is true if an AI controls the team.
	IsAI bool `json:"isAI"` //nolint:tagliatelle // the engine's name

	// Side is the team's faction as the start script holds it, e.g. "cortex", or "random". See StartUnit for the
	// faction the team played.
	Side string `json:"side"`

	// StartUnit is the unitDefID of the commander the team started with: BAR's team rules param "startUnit".
	// Is nil if the game did not set it.
	StartUnit *int `json:"startUnit,omitempty"`

	// Name is the player's name, or for an AI its name and short name, e.g. "BARbarianAI(1) (UNKNOWN)": in a playback
	// the engine gives no AI short name.
	Name string `json:"name"`
}

package events

// Team is a team other than Gaia, from Spring.GetTeamInfo at the end of the game.
type Team struct {
	// TeamID is the team's ID.
	TeamID int `json:"teamID"` //nolint:tagliatelle // the engine's name

	// Leader is the ID of the team's leading player: the start script's TeamLeader, for an AI team usually the player
	// who hosts the AI.
	Leader int `json:"leader"`

	// IsDead is true if the team died before the end of the game, see TeamDied.
	IsDead bool `json:"isDead"`

	// HasAI is true if an AI controls the team.
	HasAI bool `json:"hasAI"` //nolint:tagliatelle // the engine's name

	// Side is the team's faction as the start script holds it, e.g. "cortex", or "random". See StartUnit for the
	// faction the team played.
	Side string `json:"side"`

	// AllyTeam is the ID of the team's allyteam.
	AllyTeam int `json:"allyTeam"`

	// IncomeMultiplier is the factor the team's income is multiplied with, 1 without a bonus or handicap.
	IncomeMultiplier float64 `json:"incomeMultiplier"`

	// CustomTeamKeys provides the custom keys of the team's section in the start script, as strings.
	CustomTeamKeys map[string]string `json:"customTeamKeys"`

	// StartUnit is the unitDefID of the commander the team started with: BAR's team rules param "startUnit". The side
	// in File.Sides with this start unit is the faction the team played.
	// Is nil if the game did not set it.
	StartUnit *int `json:"startUnit,omitempty"`

	// Name is the player's name, or for an AI its name and short name, e.g. "BARbarianAI(1) (UNKNOWN)": in a playback
	// the engine gives no AI short name.
	Name string `json:"name"`
}

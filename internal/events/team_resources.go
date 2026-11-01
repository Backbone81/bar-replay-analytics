package events

// TeamResources is written for each team every 5 s: its metal and energy from Spring.GetTeamResources.
type TeamResources struct {
	// TeamID is the team's ID.
	TeamID int `json:"teamID"` //nolint:tagliatelle // the engine's name

	// Metal is the team's metal.
	Metal Resource `json:"metal"`

	// Energy is the team's energy.
	Energy Resource `json:"energy"`
}

// Resource is the state of one resource of a team, the first values Spring.GetTeamResources returns. The engine
// updates a team's resources once a second (TEAM_SLOWUPDATE_RATE in rts/Sim/Misc/GlobalConstants.h).
type Resource struct {
	// CurrentLevel is the amount in storage.
	CurrentLevel float64 `json:"currentLevel"`

	// Storage is the storage capacity.
	Storage float64 `json:"storage"`

	// Pull is the amount the team's units wanted to spend in the last second.
	Pull float64 `json:"pull"`

	// Income is the amount the team made in the last second.
	Income float64 `json:"income"`

	// Expense is the amount the team spent in the last second.
	Expense float64 `json:"expense"`
}

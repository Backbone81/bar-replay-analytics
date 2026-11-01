package events

// Resources is written for each team every 5 s: its metal and energy (Spring.GetTeamResources) and how many of its
// units, builders and factories there are and how many of them are idle.
type Resources struct {
	// TeamID is the team's ID.
	TeamID int `json:"teamID"` //nolint:tagliatelle // the engine's name

	// Metal is the team's metal.
	Metal Resource `json:"metal"`

	// Energy is the team's energy.
	Energy Resource `json:"energy"`

	// UnitCount is the number of the team's units, unfinished ones and buildings included.
	UnitCount int `json:"unitCount"`

	// BuilderCount is the number of the team's finished commanders and constructors.
	BuilderCount int `json:"builderCount"`

	// IdleBuilderCount is the number of the team's finished commanders and constructors without orders that are not
	// building.
	IdleBuilderCount int `json:"idleBuilderCount"`

	// FactoryCount is the number of the team's finished factories.
	FactoryCount int `json:"factoryCount"`

	// IdleFactoryCount is the number of the team's finished factories that are not building.
	IdleFactoryCount int `json:"idleFactoryCount"`
}

// Resource is the state of one resource of a team, as Spring.GetTeamResources returns it. The engine updates a team's
// resources once a second (TEAM_SLOWUPDATE_RATE in rts/Sim/Misc/GlobalConstants.h).
type Resource struct {
	// Current is the amount in storage.
	Current float64 `json:"current"`

	// Storage is the storage capacity.
	Storage float64 `json:"storage"`

	// Pull is the amount the team's units wanted to spend in the last second.
	Pull float64 `json:"pull"`

	// Income is the amount the team made in the last second.
	Income float64 `json:"income"`

	// Expense is the amount the team spent in the last second.
	Expense float64 `json:"expense"`
}

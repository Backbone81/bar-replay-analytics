package events

// UnitCreated is written when a unit is started (a nanoframe) or spawned (e.g. a commander at the start).
type UnitCreated struct {
	// UnitID is the unit's ID.
	UnitID int `json:"unitID"` //nolint:tagliatelle // the engine's name

	// UnitDef is the unit's def name.
	UnitDef string `json:"unitDef"`

	// TeamID is the ID of the unit's team.
	TeamID int `json:"teamID"` //nolint:tagliatelle // the engine's name

	// Position is where the unit was created.
	// Is nil if the engine gave no position.
	Position *Position `json:"position,omitempty"`

	// BuilderID is the ID of the builder or factory that started the unit.
	// Is nil if the unit was not built.
	BuilderID *int `json:"builderID,omitempty"` //nolint:tagliatelle // the engine's name

	// BuilderDef is the def name of the builder that started the unit. Is empty if the unit was not built.
	BuilderDef string `json:"builderDef,omitempty"`
}

// UnitFinished is written when a unit is finished.
type UnitFinished struct {
	// UnitID is the unit's ID.
	UnitID int `json:"unitID"` //nolint:tagliatelle // the engine's name

	// UnitDef is the unit's def name.
	UnitDef string `json:"unitDef"`

	// TeamID is the ID of the unit's team.
	TeamID int `json:"teamID"` //nolint:tagliatelle // the engine's name
}

// UnitDestroyed is written when a unit dies, also an unfinished one, a self-destructed one or one of a team that
// resigned.
type UnitDestroyed struct {
	// UnitID is the unit's ID.
	UnitID int `json:"unitID"` //nolint:tagliatelle // the engine's name

	// UnitDef is the unit's def name.
	UnitDef string `json:"unitDef"`

	// TeamID is the ID of the unit's team.
	TeamID int `json:"teamID"` //nolint:tagliatelle // the engine's name

	// Position is where the unit died.
	// Is nil if the engine gave no position.
	Position *Position `json:"position,omitempty"`

	// BuildProgress is how far the unit was built, from 0 to 1; 1 for a finished unit.
	BuildProgress float64 `json:"buildProgress"`

	// AttackerDef is the def name of the unit that killed it. Is empty without an attacker.
	AttackerDef string `json:"attackerDef,omitempty"`

	// AttackerTeamID is the ID of the attacker's team.
	// Is nil without an attacker.
	AttackerTeamID *int `json:"attackerTeamID,omitempty"` //nolint:tagliatelle // the engine's name

	// Weapon is the weapon def name of the weapon that killed it. Is empty if none did.
	Weapon string `json:"weapon,omitempty"`
}

// UnitGiven is written when a unit changes team.
type UnitGiven struct {
	// UnitID is the unit's ID.
	UnitID int `json:"unitID"` //nolint:tagliatelle // the engine's name

	// UnitDef is the unit's def name.
	UnitDef string `json:"unitDef"`

	// TeamID is the ID of the unit's new team.
	TeamID int `json:"teamID"` //nolint:tagliatelle // the engine's name

	// OldTeamID is the ID of the unit's old team.
	OldTeamID int `json:"oldTeamID"` //nolint:tagliatelle // the engine's name
}

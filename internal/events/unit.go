package events

// UnitCreated is written by widget:UnitCreated: a unit is started (a nanoframe) or spawned (e.g. a commander at the
// start).
type UnitCreated struct {
	// UnitID is the unit's ID.
	UnitID int `json:"unitID"` //nolint:tagliatelle // the engine's name

	// UnitDefID is the unit's unit def.
	UnitDefID int `json:"unitDefID"` //nolint:tagliatelle // the engine's name

	// UnitTeam is the ID of the unit's team.
	UnitTeam int `json:"unitTeam"`

	// X is the unit's x position in elmos, rounded down.
	X int `json:"x"`

	// Z is the unit's z position in elmos, rounded down.
	Z int `json:"z"`

	// BuilderID is the ID of the builder or factory that started the unit.
	// Is nil if the unit was not built.
	BuilderID *int `json:"builderID,omitempty"` //nolint:tagliatelle // the engine's name
}

// UnitFinished is written by widget:UnitFinished: a unit is finished.
type UnitFinished struct {
	// UnitID is the unit's ID.
	UnitID int `json:"unitID"` //nolint:tagliatelle // the engine's name

	// UnitDefID is the unit's unit def.
	UnitDefID int `json:"unitDefID"` //nolint:tagliatelle // the engine's name

	// UnitTeam is the ID of the unit's team.
	UnitTeam int `json:"unitTeam"`
}

// UnitDestroyed is written by widget:UnitDestroyed: a unit dies, also an unfinished one, a self-destructed one or one
// of a team that resigned.
type UnitDestroyed struct {
	// UnitID is the unit's ID.
	UnitID int `json:"unitID"` //nolint:tagliatelle // the engine's name

	// UnitDefID is the unit's unit def.
	UnitDefID int `json:"unitDefID"` //nolint:tagliatelle // the engine's name

	// UnitTeam is the ID of the unit's team.
	UnitTeam int `json:"unitTeam"`

	// X is the unit's x position in elmos, rounded down.
	X int `json:"x"`

	// Z is the unit's z position in elmos, rounded down.
	Z int `json:"z"`

	// BuildProgress is how far the unit was built, from 0 to 1; 1 for a finished unit.
	BuildProgress float64 `json:"buildProgress"`

	// AttackerID is the ID of the unit that killed it.
	// Is nil without an attacker.
	AttackerID *int `json:"attackerID,omitempty"` //nolint:tagliatelle // the engine's name

	// AttackerDefID is the unit def of the unit that killed it.
	// Is nil without an attacker.
	AttackerDefID *int `json:"attackerDefID,omitempty"` //nolint:tagliatelle // the engine's name

	// AttackerTeam is the ID of the attacker's team.
	// Is nil without an attacker.
	AttackerTeam *int `json:"attackerTeam,omitempty"`

	// WeaponDefID is the weapon def of the weapon that killed it, or negative: one of the engine's environmental damage
	// types (Game.envDamageTypes), e.g. a self-destruct.
	WeaponDefID int `json:"weaponDefID"` //nolint:tagliatelle // the engine's name
}

// UnitGiven is written by widget:UnitGiven: a unit changes team.
type UnitGiven struct {
	// UnitID is the unit's ID.
	UnitID int `json:"unitID"` //nolint:tagliatelle // the engine's name

	// UnitDefID is the unit's unit def.
	UnitDefID int `json:"unitDefID"` //nolint:tagliatelle // the engine's name

	// NewTeam is the ID of the unit's new team.
	NewTeam int `json:"newTeam"`

	// OldTeam is the ID of the unit's old team.
	OldTeam int `json:"oldTeam"`
}

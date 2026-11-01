package events

// ArmyPositions is written for each team every 30 s: where its army is, i.e. its mobile units that are not builders.
type ArmyPositions struct {
	// TeamID is the team's ID.
	TeamID int `json:"teamID"` //nolint:tagliatelle // the engine's name

	// Units provides the team's army units.
	Units []ArmyUnit `json:"units"`
}

// ArmyUnit is a unit of ArmyPositions.
type ArmyUnit struct {
	// UnitDef is the unit's def name.
	UnitDef string `json:"unitDef"`

	// Position is where the unit is.
	Position Position `json:"position"`
}

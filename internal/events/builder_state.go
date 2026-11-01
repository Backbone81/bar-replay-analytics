package events

// BuilderStateKind is what a builder does.
type BuilderStateKind string

const (
	// BuilderStateWorking means the builder has a worker task (Spring.GetUnitWorkerTask): building, repairing,
	// assisting, reclaiming or resurrecting.
	BuilderStateWorking BuilderStateKind = "working"

	// BuilderStateBusy means the builder has orders but no worker task: a mobile builder is mostly walking to its
	// work, a factory waits, e.g. for its build spot to clear.
	BuilderStateBusy BuilderStateKind = "busy"

	// BuilderStateIdle means the builder has no orders.
	BuilderStateIdle BuilderStateKind = "idle"
)

// BuilderState is written when the state of a finished commander, constructor, nano turret or factory changes. The
// widget samples it every 0.5 s.
type BuilderState struct {
	// UnitID is the builder's ID.
	UnitID int `json:"unitID"` //nolint:tagliatelle // the engine's name

	// State is what the builder does.
	State BuilderStateKind `json:"state"`

	// CmdID is the command ID of the builder's worker task while working, or of its first order while busy (for a
	// factory: of its build queue). A build is -unitDefID of what is built; otherwise 40 repair or assist, 90 reclaim,
	// 125 resurrect (CMD in Lua, rts/Sim/Units/CommandAI/Command.h).
	// Is nil while idle.
	CmdID *int `json:"cmdID,omitempty"` //nolint:tagliatelle // the engine's name

	// TargetID is the ID of the unit the builder works on, or of a feature as featureID + Game.MaxUnits.
	// Is nil unless working.
	TargetID *int `json:"targetID,omitempty"` //nolint:tagliatelle // the engine's name
}

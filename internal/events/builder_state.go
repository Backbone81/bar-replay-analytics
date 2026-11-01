package events

// BuilderStateKind is what a builder does.
type BuilderStateKind string

const (
	// BuilderStateWorking means the builder has a worker task (Spring.GetUnitWorkerTask): building, repairing,
	// assisting, reclaiming or resurrecting.
	BuilderStateWorking BuilderStateKind = "working"

	// BuilderStateBusy means the builder has orders but no worker task, mostly walking to its work.
	BuilderStateBusy BuilderStateKind = "busy"

	// BuilderStateIdle means the builder has no orders.
	BuilderStateIdle BuilderStateKind = "idle"
)

// BuilderState is written when the state of a finished commander, constructor or nano turret changes. The widget
// samples it every 0.5 s.
type BuilderState struct {
	// UnitID is the builder's ID.
	UnitID int `json:"unitID"` //nolint:tagliatelle // the engine's name

	// State is what the builder does.
	State BuilderStateKind `json:"state"`

	// Command is the engine command ID of the builder's worker task while working, or of its current order while busy:
	// negative while building, the negated unit def ID of what it builds; 40 repair or assist, 90 reclaim, 125
	// resurrect (rts/Sim/Units/CommandAI/Command.h).
	// Is nil while idle.
	Command *int `json:"command,omitempty"`

	// TargetID is the ID of the unit or feature the builder works on.
	// Is nil unless working.
	TargetID *int `json:"targetID,omitempty"` //nolint:tagliatelle // the engine's name
}

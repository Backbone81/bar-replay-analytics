package events

import "fmt"

// Event is one event of an events file. At most one of the pointers is set: the event's type. None is set for an event
// of a type this package does not know.
type Event struct {
	// Frame is the game frame the event happened at.
	Frame int `json:"frame"`

	// UnitCreated is written when a unit is started or spawned.
	UnitCreated *UnitCreated `json:"unitCreated,omitempty"`

	// UnitFinished is written when a unit is finished.
	UnitFinished *UnitFinished `json:"unitFinished,omitempty"`

	// UnitDestroyed is written when a unit dies.
	UnitDestroyed *UnitDestroyed `json:"unitDestroyed,omitempty"`

	// UnitGiven is written when a unit changes team.
	UnitGiven *UnitGiven `json:"unitGiven,omitempty"`

	// UnitIdle is written when a builder or factory has no more orders.
	UnitIdle *UnitIdle `json:"unitIdle,omitempty"`

	// UnitCommand is written when a builder or factory is given a command.
	UnitCommand *UnitCommand `json:"unitCommand,omitempty"`

	// UnitCmdDone is written when a builder or factory finished a command.
	UnitCmdDone *UnitCmdDone `json:"unitCmdDone,omitempty"`

	// TeamDied is written when a team dies.
	TeamDied *TeamDied `json:"teamDied,omitempty"`

	// GameOver is written when the game ends.
	GameOver *GameOver `json:"gameOver,omitempty"`
}

// Validate returns an error if more than one event type is set.
func (e *Event) Validate() error {
	isSet := []bool{
		e.UnitCreated != nil,
		e.UnitFinished != nil,
		e.UnitDestroyed != nil,
		e.UnitGiven != nil,
		e.UnitIdle != nil,
		e.UnitCommand != nil,
		e.UnitCmdDone != nil,
		e.TeamDied != nil,
		e.GameOver != nil,
	}
	count := 0
	for _, set := range isSet {
		if set {
			count++
		}
	}
	if count > 1 {
		return fmt.Errorf("frame %d: %d event types set, at most one is allowed", e.Frame, count)
	}
	return nil
}

// UnitDefIDs returns the unit defs the event refers to: the unit's, an attacker's, and what a build command builds.
func (e *Event) UnitDefIDs() []int {
	switch {
	case e.UnitCreated != nil:
		return []int{e.UnitCreated.UnitDefID}
	case e.UnitFinished != nil:
		return []int{e.UnitFinished.UnitDefID}
	case e.UnitDestroyed != nil:
		if e.UnitDestroyed.AttackerDefID != nil {
			return []int{e.UnitDestroyed.UnitDefID, *e.UnitDestroyed.AttackerDefID}
		}
		return []int{e.UnitDestroyed.UnitDefID}
	case e.UnitGiven != nil:
		return []int{e.UnitGiven.UnitDefID}
	case e.UnitIdle != nil:
		return []int{e.UnitIdle.UnitDefID}
	case e.UnitCommand != nil:
		return withBuild(e.UnitCommand.UnitDefID, e.UnitCommand.CmdID)
	case e.UnitCmdDone != nil:
		return withBuild(e.UnitCmdDone.UnitDefID, e.UnitCmdDone.CmdID)
	}
	return nil
}

// withBuild returns the unit's unit def, and the one a build command (cmdID < 0) builds.
func withBuild(unitDefID int, cmdID int) []int {
	if cmdID < 0 {
		return []int{unitDefID, -cmdID}
	}
	return []int{unitDefID}
}

package events

import "fmt"

// Event is one event of an events file. At most one of the pointers is set: the event's type. None is set for an event
// of a type this package does not know.
type Event struct {
	// Frame is the game frame the event happened at.
	Frame int `json:"frame"`

	// Init is the first event.
	Init *Init `json:"init,omitempty"`

	// UnitDef is written the first time a unit type shows up.
	UnitDef *UnitDef `json:"unitDef,omitempty"`

	// UnitCreated is written when a unit is started or spawned.
	UnitCreated *UnitCreated `json:"unitCreated,omitempty"`

	// UnitFinished is written when a unit is finished.
	UnitFinished *UnitFinished `json:"unitFinished,omitempty"`

	// UnitDestroyed is written when a unit dies.
	UnitDestroyed *UnitDestroyed `json:"unitDestroyed,omitempty"`

	// UnitGiven is written when a unit changes team.
	UnitGiven *UnitGiven `json:"unitGiven,omitempty"`

	// BuilderState is written when a builder's state changes.
	BuilderState *BuilderState `json:"builderState,omitempty"`

	// Resources is written for each team every few seconds.
	Resources *Resources `json:"resources,omitempty"`

	// ArmyPositions is written for each team every few seconds.
	ArmyPositions *ArmyPositions `json:"armyPositions,omitempty"`

	// GameOver is written when the game ends.
	GameOver *GameOver `json:"gameOver,omitempty"`
}

// Validate returns an error if more than one event type is set.
func (e *Event) Validate() error {
	isSet := []bool{
		e.Init != nil,
		e.UnitDef != nil,
		e.UnitCreated != nil,
		e.UnitFinished != nil,
		e.UnitDestroyed != nil,
		e.UnitGiven != nil,
		e.BuilderState != nil,
		e.Resources != nil,
		e.ArmyPositions != nil,
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

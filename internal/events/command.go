package events

// The command events are written for commanders, constructors and factories only: they tell when a builder is idle and
// what it was ordered to do. Nano turrets and other units are not followed.

// UnitIdle is written by widget:UnitIdle: a unit's command queue ran empty. The widget also writes it when a unit is
// finished or given to a team without orders, for which the engine calls no UnitIdle.
type UnitIdle struct {
	// UnitID is the unit's ID.
	UnitID int `json:"unitID"` //nolint:tagliatelle // the engine's name

	// UnitDefID is the unit's unit def.
	UnitDefID int `json:"unitDefID"` //nolint:tagliatelle // the engine's name

	// UnitTeam is the ID of the unit's team.
	UnitTeam int `json:"unitTeam"`
}

// UnitCommand is written by widget:UnitCommand: a unit was given a command, before it enters the command queue. Not
// every command does: states (fire state, move state, on/off ...), stop and remove change the queue or the unit, but
// are no orders. The command has no tag yet, so a UnitCmdDone cannot be matched to it by tag.
type UnitCommand struct {
	// UnitID is the unit's ID.
	UnitID int `json:"unitID"` //nolint:tagliatelle // the engine's name

	// UnitDefID is the unit's unit def.
	UnitDefID int `json:"unitDefID"` //nolint:tagliatelle // the engine's name

	// UnitTeam is the ID of the unit's team.
	UnitTeam int `json:"unitTeam"`

	// CmdID is the command ID (CMD in Lua, rts/Sim/Units/CommandAI/Command.h). A build is -unitDefID of what is built.
	// IDs above 150 (CMD_FAILED) are the game's own commands.
	CmdID int `json:"cmdID"` //nolint:tagliatelle // the engine's name

	// CmdParams provides the command's parameters, e.g. a target unit ID, or a position x, y, z (and facing for a
	// build).
	CmdParams []float64 `json:"cmdParams"`

	// CmdOpts is the command's option bits (CommandOptions.coded): 8 internal order, 16 right mouse button, 32 shift
	// (queued after the current orders), 64 control, 128 alt.
	CmdOpts int `json:"cmdOpts"`

	// PlayerID is the ID of the player who gave the command.
	PlayerID int `json:"playerID"` //nolint:tagliatelle // the engine's name

	// FromSynced is true if synced code gave the command, e.g. a factory to the unit it built.
	FromSynced bool `json:"fromSynced"`

	// FromLua is true if Lua code gave the command.
	FromLua bool `json:"fromLua"`
}

// UnitCmdDone is written by widget:UnitCmdDone: a unit finished a command, or it was removed from the front of its
// queue.
type UnitCmdDone struct {
	// UnitID is the unit's ID.
	UnitID int `json:"unitID"` //nolint:tagliatelle // the engine's name

	// UnitDefID is the unit's unit def.
	UnitDefID int `json:"unitDefID"` //nolint:tagliatelle // the engine's name

	// UnitTeam is the ID of the unit's team.
	UnitTeam int `json:"unitTeam"`

	// CmdID is the command ID, see UnitCommand.CmdID.
	CmdID int `json:"cmdID"` //nolint:tagliatelle // the engine's name

	// CmdParams provides the command's parameters, see UnitCommand.CmdParams.
	CmdParams []float64 `json:"cmdParams"`
}

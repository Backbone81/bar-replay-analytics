package events

// Game is the game that was played, from the engine's Lua Game table.
type Game struct {
	// MapName is the map's name, e.g. "Argent Strata 1.1".
	MapName string `json:"mapName"`

	// MapSizeX is the map's size along x in elmos.
	MapSizeX int `json:"mapSizeX"`

	// MapSizeZ is the map's size along z in elmos.
	MapSizeZ int `json:"mapSizeZ"`

	// GameName is the game's name without version, e.g. "Beyond All Reason".
	GameName string `json:"gameName"`

	// GameVersion is the game's version, e.g. "test-31720-0f97aa5".
	GameVersion string `json:"gameVersion"`

	// MaxUnits is the maximum number of units. A command's target ID (e.g. UnitCommand.CmdParams of a reclaim) of at
	// least MaxUnits is a feature: featureID + MaxUnits.
	MaxUnits int `json:"maxUnits"`
}

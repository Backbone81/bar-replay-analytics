package events

// Init is the first event: the map, the game and the teams.
type Init struct {
	// Map is the map's name (Game.mapName), e.g. "Argent Strata 1.1".
	Map string `json:"map"`

	// MapSizeX is the map's size along x in elmos.
	MapSizeX int `json:"mapSizeX"`

	// MapSizeZ is the map's size along z in elmos.
	MapSizeZ int `json:"mapSizeZ"`

	// Game is the game's name and version, e.g. "Beyond All Reason test-31756-0d4c3c3".
	Game string `json:"game"`

	// Teams provides every team but Gaia.
	Teams []Team `json:"teams"`
}

// Team is a team of Init.
type Team struct {
	// TeamID is the team's ID.
	TeamID int `json:"teamID"` //nolint:tagliatelle // the engine's name

	// AllyTeamID is the ID of the team's allyteam.
	AllyTeamID int `json:"allyTeamID"` //nolint:tagliatelle // the engine's name

	// IsAI is true if an AI controls the team.
	IsAI bool `json:"isAI"` //nolint:tagliatelle // the engine's name

	// Side is the team's faction as the start script holds it, e.g. "cortex", or "random" for an AI that chose one.
	Side string `json:"side"`

	// Name is the player's name, or for an AI its name and short name, e.g. "BARbarianAI(1) (UNKNOWN)": in a playback
	// the engine gives no AI short name.
	Name string `json:"name"`
}

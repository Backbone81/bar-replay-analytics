package events

// Side is a side of the game, as BAR calls it a faction, from Spring.GetSideData: the game's gamedata/sidedata.lua.
type Side struct {
	// SideName is the side's name in lowercase, e.g. "cortex", as Team.Side gives it.
	SideName string `json:"sideName"`

	// CaseName is the side's name as the game writes it, e.g. "Cortex".
	CaseName string `json:"caseName"`

	// StartUnit is the unit def name of the commander a team of this side starts with, in lowercase, e.g. "corcom".
	StartUnit string `json:"startUnit"`
}

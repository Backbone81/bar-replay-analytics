package events

// TeamStats is one entry of a team's statistics history, as Spring.GetTeamStatsHistory returns it at the end of the
// game: the engine records an entry every 15 s (teamStatPeriod) with the totals since the start of the game, the same
// TeamStatistics a replay file holds (rts/Sim/Misc/TeamStatistics.h). The engine records them while it simulates, so a
// playback has them also for a replay file without statistics.
type TeamStats struct {
	// Time is Frame in whole seconds.
	Time int `json:"time"`

	// Frame is the frame the entry was recorded at. The last entry holds the totals at the frame the widget read them.
	Frame int `json:"frame"`

	// MetalUsed is the metal the team has spent.
	MetalUsed float64 `json:"metalUsed"`

	// MetalProduced is the metal the team has made.
	MetalProduced float64 `json:"metalProduced"`

	// MetalExcess is the metal lost because the team's storage was full, after sharing to allies.
	MetalExcess float64 `json:"metalExcess"`

	// MetalReceived is the metal received from allies, given or shared automatically.
	MetalReceived float64 `json:"metalReceived"`

	// MetalSent is the metal sent to allies, given or shared automatically.
	MetalSent float64 `json:"metalSent"`

	// EnergyUsed is the energy the team has spent.
	EnergyUsed float64 `json:"energyUsed"`

	// EnergyProduced is the energy the team has made.
	EnergyProduced float64 `json:"energyProduced"`

	// EnergyExcess is the energy lost because the team's storage was full, after sharing to allies.
	EnergyExcess float64 `json:"energyExcess"`

	// EnergyReceived is the energy received from allies, given or shared automatically.
	EnergyReceived float64 `json:"energyReceived"`

	// EnergySent is the energy sent to allies, given or shared automatically.
	EnergySent float64 `json:"energySent"`

	// DamageDealt is the damage the team's units dealt, friendly fire included. Each hit counts at most the health its
	// target had already lost, so a hit on a full-health unit counts 0.
	DamageDealt float64 `json:"damageDealt"`

	// DamageReceived is the damage the team's units took, counted like DamageDealt.
	DamageReceived float64 `json:"damageReceived"`

	// UnitsProduced is the number of units, buildings included, the team has built.
	UnitsProduced int `json:"unitsProduced"`

	// UnitsDied is the number of the team's units that died, for any reason.
	UnitsDied int `json:"unitsDied"`

	// UnitsReceived is the number of units allies gave to the team.
	UnitsReceived int `json:"unitsReceived"`

	// UnitsSent is the number of units the team gave to allies.
	UnitsSent int `json:"unitsSent"`

	// UnitsCaptured is the number of enemy units the team captured.
	UnitsCaptured int `json:"unitsCaptured"`

	// UnitsOutCaptured is the number of the team's units enemies captured.
	UnitsOutCaptured int `json:"unitsOutCaptured"`

	// UnitsKilled is the number of enemy units the team's units killed.
	UnitsKilled int `json:"unitsKilled"`
}

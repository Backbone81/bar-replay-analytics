package demo

import (
	"encoding/binary"
	"fmt"
	"math"
)

// TeamStatistics is one entry of a team's statistics history. All values except Frame are totals since the start of
// the game.
//
// See rts/Sim/Misc/TeamStatistics.h.
type TeamStatistics struct {
	// Frame is the game frame this entry was recorded at (GAME_SPEED is 30 frames per second). The last entry is the
	// exception: it holds the totals at the end of the game, but its frame is the one the next entry would have been
	// recorded at, up to teamStatPeriod seconds after the end. See CTeam::SlowUpdate in rts/Sim/Misc/Team.cpp.
	Frame int32 `json:"frame"`

	// MetalUsed is the metal this team has spent.
	MetalUsed float32 `json:"metalUsed"`

	// EnergyUsed is the energy this team has spent.
	EnergyUsed float32 `json:"energyUsed"`

	// MetalProduced is the metal this team produced.
	MetalProduced float32 `json:"metalProduced"`

	// EnergyProduced is the energy this team produced.
	EnergyProduced float32 `json:"energyProduced"`

	// MetalExcess is the metal lost because this team's storage was full, after sharing to allies.
	MetalExcess float32 `json:"metalExcess"`

	// EnergyExcess is the energy lost because this team's storage was full, after sharing to allies.
	EnergyExcess float32 `json:"energyExcess"`

	// MetalReceived is the metal received from allies, given or shared automatically.
	MetalReceived float32 `json:"metalReceived"`

	// EnergyReceived is the energy received from allies, given or shared automatically.
	EnergyReceived float32 `json:"energyReceived"`

	// MetalSent is the metal sent to allies, given or shared automatically.
	MetalSent float32 `json:"metalSent"`

	// EnergySent is the energy sent to allies, given or shared automatically.
	EnergySent float32 `json:"energySent"`

	// DamageDealt is the damage this team's units dealt, to any unit, friendly fire included. Each hit counts at most
	// the health its target had already lost, so a hit on a full-health unit counts 0.
	DamageDealt float32 `json:"damageDealt"`

	// DamageReceived is the damage this team's units took, counted like DamageDealt, also from hits without an
	// attacker.
	DamageReceived float32 `json:"damageReceived"`

	// UnitsProduced is the number of units this team built, buildings included.
	UnitsProduced int32 `json:"unitsProduced"`

	// UnitsDied is the number of this team's units that died, for any reason.
	UnitsDied int32 `json:"unitsDied"`

	// UnitsReceived is the number of units other teams gave to this team.
	UnitsReceived int32 `json:"unitsReceived"`

	// UnitsSent is the number of units this team gave to other teams.
	UnitsSent int32 `json:"unitsSent"`

	// UnitsCaptured is the number of units this team captured from enemies.
	UnitsCaptured int32 `json:"unitsCaptured"`

	// UnitsOutCaptured is the number of this team's units captured by enemies.
	UnitsOutCaptured int32 `json:"unitsOutCaptured"`

	// UnitsKilled is the number of finished enemy units this team's units have killed, collisions not included.
	UnitsKilled int32 `json:"unitsKilled"`
}

// ReadTeamStatistics reads and validates a TeamStatistics from the start of data.
func ReadTeamStatistics(data []byte) (TeamStatistics, error) {
	var stats TeamStatistics
	if _, err := binary.Decode(data, ByteOrder, &stats); err != nil {
		return TeamStatistics{}, fmt.Errorf("reading TeamStatistics: %w", err)
	}
	if err := stats.validate(); err != nil {
		return TeamStatistics{}, fmt.Errorf("invalid TeamStatistics: %w", err)
	}
	return stats, nil
}

func (s *TeamStatistics) validate() error {
	if s.Frame < 0 {
		return fmt.Errorf("frame: got %d, want >= 0", s.Frame)
	}

	floats := []struct {
		name  string
		value float32
	}{
		{"metalUsed", s.MetalUsed},
		{"energyUsed", s.EnergyUsed},
		{"metalProduced", s.MetalProduced},
		{"energyProduced", s.EnergyProduced},
		{"metalExcess", s.MetalExcess},
		{"energyExcess", s.EnergyExcess},
		{"metalReceived", s.MetalReceived},
		{"energyReceived", s.EnergyReceived},
		{"metalSent", s.MetalSent},
		{"energySent", s.EnergySent},
		{"damageDealt", s.DamageDealt},
		{"damageReceived", s.DamageReceived},
	}
	for _, f := range floats {
		if math.IsNaN(float64(f.value)) || math.IsInf(float64(f.value), 0) {
			return fmt.Errorf("%s: got %v, want a finite number", f.name, f.value)
		}
	}

	ints := []struct {
		name  string
		value int32
	}{
		{"unitsProduced", s.UnitsProduced},
		{"unitsDied", s.UnitsDied},
		{"unitsReceived", s.UnitsReceived},
		{"unitsSent", s.UnitsSent},
		{"unitsCaptured", s.UnitsCaptured},
		{"unitsOutCaptured", s.UnitsOutCaptured},
		{"unitsKilled", s.UnitsKilled},
	}
	for _, i := range ints {
		if i.value < 0 {
			return fmt.Errorf("%s: got %d, want >= 0", i.name, i.value)
		}
	}
	return nil
}

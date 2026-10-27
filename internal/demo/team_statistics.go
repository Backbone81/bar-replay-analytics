package demo

import (
	"encoding/binary"
	"fmt"
	"math"
)

// TeamStatistics is one entry of a team's statistics history. All values except Frame are cumulative since the start
// of the game.
//
// Defined in the Recoil engine source: rts/Sim/Misc/TeamStatistics.h.
type TeamStatistics struct {
	// Frame is the game frame this entry was recorded at (GAME_SPEED is 30 frames per second).
	Frame int32

	// MetalUsed is the metal this team has spent.
	MetalUsed float32

	// EnergyUsed is the energy this team has spent.
	EnergyUsed float32

	// MetalProduced is this team's metal income.
	MetalProduced float32

	// EnergyProduced is this team's energy income.
	EnergyProduced float32

	// MetalExcess is the metal lost because this team's storage was full, after sharing to allies.
	MetalExcess float32

	// EnergyExcess is the energy lost because this team's storage was full, after sharing to allies.
	EnergyExcess float32

	// MetalReceived is the metal received from allies, given or shared automatically.
	MetalReceived float32

	// EnergyReceived is the energy received from allies, given or shared automatically.
	EnergyReceived float32

	// MetalSent is the metal sent to allies, given or shared automatically.
	MetalSent float32

	// EnergySent is the energy sent to allies, given or shared automatically.
	EnergySent float32

	// DamageDealt is the damage this team's units dealt, to any unit, friendly fire included. Each hit counts at most
	// the health its target had already lost, so a hit on a full-health unit counts 0.
	DamageDealt float32

	// DamageReceived is the damage this team's units took, counted like DamageDealt, also from hits without an
	// attacker.
	DamageReceived float32

	// UnitsProduced is the number of units, buildings included, this team has built.
	UnitsProduced int32

	// UnitsDied is the number of this team's units that died, for any reason.
	UnitsDied int32

	// UnitsReceived is the number of units other teams gave to this team.
	UnitsReceived int32

	// UnitsSent is the number of units this team gave to other teams.
	UnitsSent int32

	// UnitsCaptured is the number of units this team captured from enemies.
	UnitsCaptured int32

	// UnitsOutCaptured is the number of this team's units captured by enemies.
	UnitsOutCaptured int32

	// UnitsKilled is the number of finished enemy units this team's units have killed, collisions not included.
	UnitsKilled int32
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

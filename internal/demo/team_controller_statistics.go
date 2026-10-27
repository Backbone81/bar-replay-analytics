package demo

import (
	"encoding/binary"
	"fmt"
)

// TeamControllerStatistics are the statistics shared by players and AIs. PlayerStatistics derives from it.
//
// Defined in the Recoil engine source: rts/Game/Players/TeamController.h.
type TeamControllerStatistics struct {
	// NumCommands is the total amount of commands issued.
	NumCommands int32

	// UnitCommands is the total amount of units affected by commands. Divide by NumCommands for average
	// units/command.
	UnitCommands int32
}

// ReadTeamControllerStatistics reads and validates a TeamControllerStatistics from the start of data.
func ReadTeamControllerStatistics(data []byte) (TeamControllerStatistics, error) {
	var stats TeamControllerStatistics
	if _, err := binary.Decode(data, ByteOrder, &stats); err != nil {
		return TeamControllerStatistics{}, fmt.Errorf("reading TeamControllerStatistics: %w", err)
	}
	if err := stats.validate(); err != nil {
		return TeamControllerStatistics{}, fmt.Errorf("invalid TeamControllerStatistics: %w", err)
	}
	return stats, nil
}

func (s *TeamControllerStatistics) validate() error {
	if s.NumCommands < 0 {
		return fmt.Errorf("numCommands: got %d, want >= 0", s.NumCommands)
	}

	if s.UnitCommands < 0 {
		return fmt.Errorf("unitCommands: got %d, want >= 0", s.UnitCommands)
	}
	return nil
}

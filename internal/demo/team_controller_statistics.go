package demo

import (
	"encoding/binary"
	"fmt"
)

// TeamControllerStatistics are the statistics players and AIs share; PlayerStatistics embeds them.
//
// See rts/Game/Players/TeamController.h.
type TeamControllerStatistics struct {
	// NumCommands is the number of commands given.
	NumCommands int32 `json:"numCommands"`

	// UnitCommands is the number of units the commands went to, summed over all commands. Divide by NumCommands for
	// units per command.
	UnitCommands int32 `json:"unitCommands"`
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

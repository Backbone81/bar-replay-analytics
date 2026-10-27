package demo

import (
	"encoding/binary"
	"fmt"
)

// PlayerStatistics contains statistical data about a player concerning a single game.
//
// See rts/Game/Players/PlayerStatistics.h.
type PlayerStatistics struct {
	TeamControllerStatistics

	// MousePixels reports how many pixels the mouse has traversed in total.
	MousePixels int32

	// MouseClicks reports how many mouse clicks happened in total.
	MouseClicks int32

	// KeyPresses reports how many keyboard presses happened in total.
	KeyPresses int32
}

// ReadPlayerStatistics reads and validates a PlayerStatistics from the start of data.
func ReadPlayerStatistics(playerStatsData []byte) (PlayerStatistics, error) {
	var stats PlayerStatistics
	if _, err := binary.Decode(playerStatsData, ByteOrder, &stats); err != nil {
		return PlayerStatistics{}, fmt.Errorf("reading PlayerStatistics: %w", err)
	}
	if err := stats.validate(); err != nil {
		return PlayerStatistics{}, fmt.Errorf("invalid PlayerStatistics: %w", err)
	}
	return stats, nil
}

func (s *PlayerStatistics) validate() error {
	if err := s.TeamControllerStatistics.validate(); err != nil {
		return err
	}

	if s.MousePixels < 0 {
		return fmt.Errorf("mousePixels: got %d, want >= 0", s.MousePixels)
	}

	if s.MouseClicks < 0 {
		return fmt.Errorf("mouseClicks: got %d, want >= 0", s.MouseClicks)
	}

	if s.KeyPresses < 0 {
		return fmt.Errorf("keyPresses: got %d, want >= 0", s.KeyPresses)
	}
	return nil
}

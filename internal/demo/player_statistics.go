package demo

import (
	"encoding/binary"
	"fmt"
)

// PlayerStatistics are a player's command and input counts for the game.
//
// See rts/Game/Players/PlayerStatistics.h.
type PlayerStatistics struct {
	TeamControllerStatistics

	// MousePixels is the distance the mouse moved, in pixels. It can be negative and is therefore not validated:
	// CMouseHandler::MouseMove adds (int)sqrt(float(dx*dx + dy*dy)) for every mouse motion event, so a single
	// bogus event with a delta above about 46340 pixels overflows the int sum, the square root of the negative
	// value is NaN, and casting NaN to int on x86 adds INT_MIN (seen for a spectator in a 2026.07.04 demo).
	MousePixels int32 `json:"mousePixels"`

	// MouseClicks is the number of mouse clicks.
	MouseClicks int32 `json:"mouseClicks"`

	// KeyPresses is the number of key presses.
	KeyPresses int32 `json:"keyPresses"`
}

// ReadPlayerStatistics reads and validates a PlayerStatistics from the start of data.
func ReadPlayerStatistics(playerStatsData []byte) (PlayerStatistics, error) {
	return readPlayerStatisticsInOrder(playerStatsData, ByteOrder)
}

// readPlayerStatisticsInOrder reads and validates a PlayerStatistics in byteOrder from the start of data.
func readPlayerStatisticsInOrder(playerStatsData []byte, byteOrder binary.ByteOrder) (PlayerStatistics, error) {
	var stats PlayerStatistics
	if _, err := binary.Decode(playerStatsData, byteOrder, &stats); err != nil {
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

	if s.MouseClicks < 0 {
		return fmt.Errorf("mouseClicks: got %d, want >= 0", s.MouseClicks)
	}

	if s.KeyPresses < 0 {
		return fmt.Errorf("keyPresses: got %d, want >= 0", s.KeyPresses)
	}
	return nil
}

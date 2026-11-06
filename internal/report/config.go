package report

import (
	"fmt"
	"time"
)

// Config configures NewReport. NewReport starts from DefaultConfig and applies its options to it.
//
// The report aggregates the game in intervals: of OpeningIntervalLength during the first OpeningLength of the game,
// where economies grow fastest, and of IntervalLength after that. The interval lengths are rounded up to whole
// statistics periods (demo.DemoFileHeader.TeamStatPeriod). The boundaries do not depend on the game, so the intervals
// of different games can be compared.
type Config struct {
	// OpeningLength is the length of the opening, which the report aggregates in intervals of OpeningIntervalLength.
	// No interval of the opening runs past its end. Zero means no opening. Must not be negative.
	OpeningLength time.Duration

	// OpeningIntervalLength is the length of the intervals in the opening. Must be positive if OpeningLength is.
	OpeningIntervalLength time.Duration

	// IntervalLength is the length of the intervals after the opening. Must be positive.
	IntervalLength time.Duration

	// OpeningBuildOrderLength is the length of the opening build order (Player.OpeningBuildOrder): it lists what was
	// finished before then. It does not depend on OpeningLength. Zero means no opening build order. Must not be
	// negative.
	OpeningBuildOrderLength time.Duration

	// IdleStretchLength is the shortest idle stretch of a builder Builder.IdleStretches lists. Must be positive.
	IdleStretchLength time.Duration

	// MapGridCellSize is the longest side of a cell of the MapGrid, in elmos. Must be positive.
	MapGridCellSize int
}

// DefaultConfig is the configuration NewReport starts from.
//
// The opening is 10 minutes in intervals of 1 minute (4 statistics periods of 15 s): the economy build-up before
// tech 2. After it, intervals of 5 minutes are enough to follow the economy.
//
// OpeningBuildOrderLength is 5 minutes: by then the commander's first queue, the first factory and the first
// constructors are done; what follows is mostly more of the same, which the economy intervals show better.
//
// IdleStretchLength is 1 minute: shorter gaps between orders are normal, a builder idle longer was forgotten.
//
// MapGridCellSize is 2048 elmos (4 of the lobby's map size units of 512 elmos): the reach of one or two defences, more
// than a cluster of metal spots, less than a base. BAR's maps are 3072 to 16384 elmos along a side, so a grid has 2 to
// 8 columns and rows; a 1v1 map of 8192 x 8192 has 4 x 4.
var DefaultConfig = Config{
	OpeningLength:           10 * time.Minute,
	OpeningIntervalLength:   time.Minute,
	IntervalLength:          5 * time.Minute,
	OpeningBuildOrderLength: 5 * time.Minute,
	IdleStretchLength:       time.Minute,
	MapGridCellSize:         2048,
}

// Validate returns an error if a length of the configuration is invalid.
func (c Config) Validate() error {
	if c.OpeningLength < 0 {
		return fmt.Errorf("opening length: got %v, want >= 0", c.OpeningLength)
	}
	if c.OpeningLength > 0 && c.OpeningIntervalLength <= 0 {
		return fmt.Errorf("opening interval length: got %v, want > 0", c.OpeningIntervalLength)
	}
	if c.IntervalLength <= 0 {
		return fmt.Errorf("interval length: got %v, want > 0", c.IntervalLength)
	}
	if c.OpeningBuildOrderLength < 0 {
		return fmt.Errorf("opening build order length: got %v, want >= 0", c.OpeningBuildOrderLength)
	}
	if c.IdleStretchLength <= 0 {
		return fmt.Errorf("idle stretch length: got %v, want > 0", c.IdleStretchLength)
	}
	if c.MapGridCellSize <= 0 {
		return fmt.Errorf("map grid cell size: got %d, want > 0", c.MapGridCellSize)
	}
	return nil
}

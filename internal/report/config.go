package report

import (
	"fmt"
	"time"
)

// Config configures NewReport. NewReport starts from DefaultConfig and applies its options to it.
//
// The report aggregates the game in intervals: of OpeningIntervalLength during the first OpeningLength of the game,
// where economies grow fastest and build orders differ most, and of IntervalLength after that. The interval lengths
// are rounded up to whole statistics periods (demo.DemoFileHeader.TeamStatPeriod). The boundaries depend only on the
// configuration, not on the game, so the intervals of different games can be compared.
type Config struct {
	// OpeningLength is the length of the opening, which the report aggregates in intervals of OpeningIntervalLength.
	// No interval of the opening runs past its end. Zero means no opening. Must not be negative. Defaults to 10
	// minutes.
	OpeningLength time.Duration

	// OpeningIntervalLength is the length of the intervals in the opening. Must be positive if OpeningLength is.
	// Defaults to 1 minute.
	OpeningIntervalLength time.Duration

	// IntervalLength is the length of the intervals after the opening. Must be positive. Defaults to 5 minutes.
	IntervalLength time.Duration
}

// DefaultConfig is the configuration NewReport starts from.
var DefaultConfig = Config{
	OpeningLength:         10 * time.Minute,
	OpeningIntervalLength: time.Minute,
	IntervalLength:        5 * time.Minute,
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
	return nil
}

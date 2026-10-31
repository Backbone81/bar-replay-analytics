package report

import "time"

// Config configures NewReport. NewReport starts from DefaultConfig and applies its options to it.
type Config struct {
	// IntervalLength is the length of the intervals the report aggregates the game in, rounded up to whole statistics
	// periods (demo.DemoFileHeader.TeamStatPeriod). Must be positive. Defaults to 5 minutes.
	IntervalLength time.Duration
}

// DefaultConfig is the configuration NewReport starts from.
var DefaultConfig = Config{
	IntervalLength: 5 * time.Minute,
}

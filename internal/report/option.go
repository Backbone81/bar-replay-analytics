package report

import "time"

// Option changes the Config NewReport works with.
type Option func(config *Config)

// WithOpeningLength sets Config.OpeningLength.
func WithOpeningLength(openingLength time.Duration) Option {
	return func(config *Config) {
		config.OpeningLength = openingLength
	}
}

// WithOpeningIntervalLength sets Config.OpeningIntervalLength.
func WithOpeningIntervalLength(openingIntervalLength time.Duration) Option {
	return func(config *Config) {
		config.OpeningIntervalLength = openingIntervalLength
	}
}

// WithIntervalLength sets Config.IntervalLength.
func WithIntervalLength(intervalLength time.Duration) Option {
	return func(config *Config) {
		config.IntervalLength = intervalLength
	}
}

// WithOpeningBuildOrderLength sets Config.OpeningBuildOrderLength.
func WithOpeningBuildOrderLength(openingBuildOrderLength time.Duration) Option {
	return func(config *Config) {
		config.OpeningBuildOrderLength = openingBuildOrderLength
	}
}

// WithIdleStretchLength sets Config.IdleStretchLength.
func WithIdleStretchLength(idleStretchLength time.Duration) Option {
	return func(config *Config) {
		config.IdleStretchLength = idleStretchLength
	}
}

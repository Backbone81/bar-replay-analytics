package report

import "time"

// Option changes the Config NewReport works with.
type Option func(config *Config)

// WithIntervalLength sets Config.IntervalLength.
func WithIntervalLength(intervalLength time.Duration) Option {
	return func(config *Config) {
		config.IntervalLength = intervalLength
	}
}

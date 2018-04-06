package stats

import (
	"time"
)

// Stats is an interface for app statistics
type Stats interface {
	Start()
	Stop()

	// Count is a simple counter
	Count(key string, n interface{}, meta ...map[string]string)
	// Inc increments the given counter by 1
	Inc(key string, meta ...map[string]string)
	// Dec decrements the given counter by 1
	Dec(key string, meta ...map[string]string)
	// Gauge measures the amount, level, or contents of something
	// The given value replaces the current one
	// e.g. in-flight requests, uptime, ...
	Gauge(key string, n interface{}, meta ...map[string]string)
	// Timing measures how long it takes to accomplish something
	// e.g. algorithm, request, ...
	Timing(key string, t time.Duration, meta ...map[string]string)
	// Histogram measures the distribution of values over the time
	Histogram(key string, n interface{}, tags ...map[string]string)
}

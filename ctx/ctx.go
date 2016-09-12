package ctx

import "github.com/stairlin/lego/stats"

// Ctx is the root interface that defines a context
type Ctx interface {
	Logger
	Stats
}

// Logger provides the core interface for logging
type Logger interface {
	Trace(tag string, args ...interface{})
	Tracef(tag string, format string, args ...interface{})
	Warning(args ...interface{})
	Warningf(format string, args ...interface{})
	Error(args ...interface{})
	Errorf(format string, args ...interface{})
}

// Stats provides the core interface for stats
type Stats interface {
	Stats() stats.Stats
}

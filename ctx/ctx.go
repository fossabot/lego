package ctx

import "github.com/stairlin/lego/stats"

// Ctx is the root interface that defines a context
type Ctx interface {
	Logger
	Stats
}

// Logger provides the core interface for logging
type Logger interface {
	Debug(tag string, args ...interface{})
	Debugf(tag string, format string, args ...interface{})
	Info(tag string, args ...interface{})
	Infof(tag string, format string, args ...interface{})
	Warning(args ...interface{})
	Warningf(format string, args ...interface{})
	Error(args ...interface{})
	Errorf(format string, args ...interface{})
}

// Stats provides the core interface for stats
type Stats interface {
	Stats() stats.Stats
}

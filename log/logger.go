package log

import "fmt"

// Level defines log severity
type Level int

// ParseLevel parses a string representation of a log level
func ParseLevel(s string) Level {
	switch s {
	case "trace":
		return LevelTrace
	case "warning":
		return LevelWarning
	case "error":
		return LevelError
	}
	return LevelTrace
}

const (
	// LevelTrace displays logs with trace level (and above)
	LevelTrace Level = iota
	// LevelWarning displays logs with warning level (and above)
	LevelWarning
	// LevelError displays only logs with error level
	LevelError
)

// String returns a string representation of the given level
func (l Level) String() string {
	switch l {
	case LevelTrace:
		return "TR"
	case LevelWarning:
		return "WN"
	case LevelError:
		return "ER"
	default:
		panic(fmt.Sprintf("unknown level <%d>", l))
	}
}

// Logger is an interface for app loggers
type Logger interface {
	// Trace level logs are to follow the code executio step by step
	Trace(tag, msg string, fields ...Field)
	// Warning level logs are meant to draw attention above a certain threshold
	// e.g. wrong credentials, 404 status code returned, upstream node down
	Warning(tag, msg string, fields ...Field)
	// Error level logs need immediate attention
	// The 2AM rule applies here, which means that if you are on call, this log line will wake you up at 2AM
	// e.g. all critical upstream nodes are down, disk space is full
	Error(tag, msg string, fields ...Field)

	// With returns a child logger, and optionally add some context to that logger
	With(fields ...Field) Logger

	// AddCalldepth adds the given value to calldepth
	// Calldepth is the count of the number of
	// frames to skip when computing the file name and line number
	AddCalldepth(n int) Logger
}

// Formatter converts a log line to a specific format, such as JSON
type Formatter interface {
	// Format formats the given log line
	Format(ctx *Ctx, tag, msg string, fields ...Field) (string, error)
}

// Printer outputs a log line somewhere, such as stdout, syslog, 3rd party service
type Printer interface {
	// Print prints the given log line
	Print(ctx *Ctx, s string) error
}

// Ctx carries the log line context (level, timestamp, ...)
type Ctx struct {
	Level     string
	Timestamp string
	Service   string
	File      string
}

package log

// Level defines log severity
type Level int

const (
	// LevelTrace displays logs with trace level (and above)
	LevelTrace Level = iota
	// LevelWarning displays logs with warning level (and above)
	LevelWarning
	// LevelError displays only logs with error level
	LevelError
)

// Logger is an interface for app loggers
type Logger interface {
	Trace(args ...interface{})
	Traceln(args ...interface{})
	Tracef(format string, args ...interface{})
	Warning(args ...interface{})
	Warningln(args ...interface{})
	Warningf(format string, args ...interface{})
	Error(args ...interface{})
	Errorln(args ...interface{})
	Errorf(format string, args ...interface{})
}

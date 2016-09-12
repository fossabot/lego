package log

// Level defines log severity
type Level int

const (
	// LevelTrace displays logs with trace level (and above)
	LevelTrace Level = iota
	// LevelInfo displays logs with info level (and above)
	LevelInfo
	// LevelWarning displays logs with warning level (and above)
	LevelWarning
	// LevelError displays only logs with error level
	LevelError
)

// Logger is an interface for app loggers
type Logger interface {
	Info(args ...interface{})
	Infoln(args ...interface{})
	Infof(format string, args ...interface{})
	Warning(args ...interface{})
	Warningln(args ...interface{})
	Warningf(format string, args ...interface{})
	Error(args ...interface{})
	Errorln(args ...interface{})
	Errorf(format string, args ...interface{})
}

package log

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/golang/glog"
)

// StdLogger outputs logs in stdout
type StdLogger struct {
	L Level
}

var (
	traceColour   = color.New(color.FgCyan)
	infoColour    = color.New(color.FgBlue)
	warningColour = color.New(color.FgYellow)
	errColour     = color.New(color.FgRed)
)

// NewStdLogger creates a new logger with the given level
func NewStdLogger(l string) (Logger, error) {
	switch l {
	case "trace":
		return &StdLogger{L: LevelTrace}, nil
	case "info":
		return &StdLogger{L: LevelInfo}, nil
	case "warning":
		return &StdLogger{L: LevelWarning}, nil
	case "error":
		return &StdLogger{L: LevelError}, nil
	}

	return nil, fmt.Errorf("unknown level <%s>", l)
}

func (l *StdLogger) Info(args ...interface{}) {
	if l.L <= LevelInfo {
		glog.Info(infoColour.SprintFunc()(args...))
	}
}

func (l *StdLogger) Infoln(args ...interface{}) {
	if l.L <= LevelInfo {
		glog.Info(infoColour.SprintFunc()(args...))
	}
}

func (l *StdLogger) Infof(format string, args ...interface{}) {
	if l.L <= LevelInfo {
		glog.Info(infoColour.SprintfFunc()(format, args...))
	}
}

func (l *StdLogger) Warning(args ...interface{}) {
	if l.L <= LevelWarning {
		glog.Warning(warningColour.SprintFunc()(args...))
	}
}

func (l *StdLogger) Warningln(args ...interface{}) {
	if l.L <= LevelWarning {
		glog.Warningln(warningColour.SprintFunc()(args...))
	}
}

func (l *StdLogger) Warningf(format string, args ...interface{}) {
	if l.L <= LevelWarning {
		glog.Warningf(warningColour.SprintfFunc()(format, args...))
	}
}

func (l *StdLogger) Error(args ...interface{}) {
	if l.L <= LevelError {
		glog.Error(errColour.SprintFunc()(args...))
	}
}

func (l *StdLogger) Errorln(args ...interface{}) {
	if l.L <= LevelError {
		glog.Errorln(errColour.SprintFunc()(args...))
	}
}

func (l *StdLogger) Errorf(format string, args ...interface{}) {
	if l.L <= LevelError {
		glog.Errorf(errColour.SprintfFunc()(format, args...))
	}
}

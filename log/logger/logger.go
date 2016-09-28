package logger

import (
	"fmt"
	"runtime"
	"time"

	"github.com/stairlin/lego/config"
	"github.com/stairlin/lego/log"
	"github.com/stairlin/lego/log/formatter"
	"github.com/stairlin/lego/log/printer"
)

// New creates a new logger
func New(service string, config *config.Log) (log.Logger, error) {
	f, err := formatter.New(config)
	if err != nil {
		return nil, err
	}

	p, err := printer.New(config)
	if err != nil {
		return nil, err
	}

	return &Logger{service: service, fmt: f, pnt: p, calldepth: 1}, nil
}

type Logger struct {
	service   string
	level     log.Level
	fmt       log.Formatter
	pnt       log.Printer
	calldepth int

	fields []log.Field
}

func (l *Logger) Trace(tag, msg string, fields ...log.Field) {
	l.log(log.LevelTrace, tag, msg, fields...)
}

func (l *Logger) Warning(tag, msg string, fields ...log.Field) {
	l.log(log.LevelWarning, "", msg, fields...)
}

func (l *Logger) Error(tag, msg string, fields ...log.Field) {
	l.log(log.LevelError, "", msg, fields...)
}

func (l *Logger) With(fields ...log.Field) log.Logger {
	c := l.clone()
	c.fields = append(c.fields, fields...)
	return c
}

func (l *Logger) AddCalldepth(n int) log.Logger {
	c := l.clone()
	c.calldepth = c.calldepth + n
	return c
}

func (l *Logger) clone() *Logger {
	return &Logger{
		service:   l.service,
		level:     l.level,
		fmt:       l.fmt,
		pnt:       l.pnt,
		fields:    l.fields,
		calldepth: l.calldepth,
	}
}

func (l *Logger) log(lvl log.Level, tag, msg string, fields ...log.Field) {
	if l.level > lvl {
		return
	}

	// Get file and line number
	_, file, line, ok := runtime.Caller(l.calldepth + 1)
	if ok {
		short := file
		for i := len(file) - 1; i > 0; i-- {
			if file[i] == '/' {
				short = file[i+1:]
				break
			}
		}
		file = short
	} else {
		file = "???"
		line = 0
	}

	ctx := log.Ctx{
		Level:     lvl.String(),
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Service:   l.service,
		File:      fmt.Sprintf("%s:%d", file, line),
	}

	f, err := l.fmt.Format(&ctx, tag, msg, fields...)
	if err != nil {
		f = fmt.Sprintf("log formatter error <%s>", err)
	}

	l.pnt.Print(&ctx, f)
}

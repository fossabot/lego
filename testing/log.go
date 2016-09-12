package testing

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stairlin/lego/log"
)

const (
	// T is the TRACE log constant
	TC = "TRACE"
	// W is the WARNING log constant
	WN = "WARN"
	// E is the ERROR log constant
	ER = "ERRR"
)

// Logger is a simple Logger interface useful for tests
type Logger struct {
	mu sync.RWMutex
	t  *testing.T

	lines map[string]int
}

// NewLogger creates a new logger
func NewLogger(t *testing.T) log.Logger {
	return &Logger{
		t:     t,
		lines: map[string]int{},
	}
}

func (l *Logger) l(s string, args ...interface{}) {
	l.t.Log(s, fmt.Sprint(args...))
	l.inc(s)
}

func (l *Logger) lf(s, format string, args ...interface{}) {
	l.t.Log(s, fmt.Sprintf(format, args...))
	l.inc(s)
}

func (l *Logger) inc(s string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.lines[s]++
}

// Lines returns the number of log lines for the given severity
func (l *Logger) Lines(s string) int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.lines[s]
}

func (l *Logger) Trace(args ...interface{})              { l.l(TC, args...) }
func (l *Logger) Traceln(args ...interface{})            { l.l(TC, args...) }
func (l *Logger) Tracef(f string, args ...interface{})   { l.lf(TC, f, args...) }
func (l *Logger) Warning(args ...interface{})            { l.l(WN, args...) }
func (l *Logger) Warningln(args ...interface{})          { l.l(WN, args...) }
func (l *Logger) Warningf(f string, args ...interface{}) { l.lf(WN, f, args...) }
func (l *Logger) Error(args ...interface{})              { l.l(ER, args...) }
func (l *Logger) Errorln(args ...interface{})            { l.l(ER, args...) }
func (l *Logger) Errorf(f string, args ...interface{})   { l.lf(ER, f, args...) }

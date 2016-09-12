package testing

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stairlin/lego/log"
)

const (
	// I is the INFO log constant
	I = "INFO"
	// W is the WARNING log constant
	W = "WARN"
	// E is the ERROR log constant
	E = "ERRR"
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

func (l *Logger) Info(args ...interface{})               { l.l(I, args...) }
func (l *Logger) Infoln(args ...interface{})             { l.l(I, args...) }
func (l *Logger) Infof(f string, args ...interface{})    { l.lf(I, f, args...) }
func (l *Logger) Warning(args ...interface{})            { l.l(W, args...) }
func (l *Logger) Warningln(args ...interface{})          { l.l(W, args...) }
func (l *Logger) Warningf(f string, args ...interface{}) { l.lf(W, f, args...) }
func (l *Logger) Error(args ...interface{})              { l.l(E, args...) }
func (l *Logger) Errorln(args ...interface{})            { l.l(E, args...) }
func (l *Logger) Errorf(f string, args ...interface{})   { l.lf(E, f, args...) }

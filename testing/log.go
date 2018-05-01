package testing

import (
	"bytes"
	"sync"
	"testing"

	"github.com/stairlin/lego/log"
)

const (
	// TC is the TRACE log constant
	TC = "TRACE"
	// WN is the WARNING log constant
	WN = "WARN"
	// ER is the ERROR log constant
	ER = "ERRR"
)

// Logger is a simple Logger interface useful for tests
type Logger struct {
	mu sync.RWMutex
	t  *testing.T

	calldepth int
	lines     map[string]int
	fields    []log.Field
	strict    bool
}

// NewLogger creates a new logger
func NewLogger(t *testing.T, strict bool) log.Logger {
	return &Logger{
		t:         t,
		calldepth: 1,
		lines:     map[string]int{},
		strict:    strict,
	}
}

func (l *Logger) l(s, tag, msg string, args ...log.Field) {
	l.t.Log(s, format(tag, msg, args...))
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

func (l *Logger) Trace(tag, msg string, fields ...log.Field)   { l.l(TC, tag, msg, fields...) }
func (l *Logger) Warning(tag, msg string, fields ...log.Field) { l.l(WN, tag, msg, fields...) }
func (l *Logger) Error(tag, msg string, fields ...log.Field) {
	l.l(ER, tag, msg, fields...)

	if l.strict {
		l.t.Error(format(tag, msg, fields...)) // Make the tests fail
	}
}
func (l *Logger) With(fields ...log.Field) log.Logger {
	nl := NewLogger(l.t, l.strict).(*Logger)
	nl.fields = append(l.fields, fields...)
	return nl
}
func (l *Logger) AddCalldepth(n int) log.Logger {
	nl := NewLogger(l.t, l.strict).(*Logger)
	nl.calldepth = nl.calldepth + n
	return nl
}
func (l *Logger) Close() error {
	return nil
}

func format(tag, msg string, fields ...log.Field) string {
	var b bytes.Buffer

	b.WriteString(tag)
	b.WriteString(" ")
	b.WriteString(msg)
	b.WriteString(" ")

	for _, f := range fields {
		k, v := f.KV()
		b.WriteString(k)
		b.WriteString("=")
		b.WriteString(v)
		b.WriteString(" ")
	}
	return b.String()
}

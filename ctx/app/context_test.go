package app_test

import (
	"testing"

	"github.com/stairlin/lego/log"
	lt "github.com/stairlin/lego/testing"
)

func TestLogger(t *testing.T) {
	tt := lt.New(t)
	app := tt.NewAppCtx("journey-test")
	logger := app.L().(*lt.Logger)

	// Send a few log lines
	app.Trace("j.test.trace", "A trace line")
	app.Trace("j.test.trace", "A second trace line")
	app.Trace("j.test.trace", "A third trace line")
	app.Trace("j.test.trace", "A fourth trace line")
	app.Warning("j.test.warning", "A warning line")
	app.Warning("j.test.warning", "Another warning line")
	app.Warning("j.test.warning", "Yet another warning line")
	app.Error("j.test.error", "An error line")
	app.Error("j.test.error", "Another error line")

	tests := []struct {
		level    string
		expected int
	}{
		{level: lt.TC, expected: 4},
		{level: lt.WN, expected: 3},
		{level: lt.ER, expected: 2},
	}

	// Ensure they have been sent to the logger
	for _, test := range tests {
		res := logger.Lines(test.level)
		if res != test.expected {
			tt.Errorf(
				"expect logger to receive %d log lines for level <%s>, but got %d",
				test.expected,
				test.level,
				res,
			)
		}
	}
}

func TestLogLevelStats(t *testing.T) {
	tt := lt.New(t)
	app := tt.NewAppCtx("journey-test")
	stats := app.Stats().(*lt.Stats)
	key := "log.level"

	// Send a few log lines
	app.Trace("j.test.trace", "A trace line")
	app.Trace("j.test.trace", "A second trace line")
	app.Trace("j.test.trace", "A third trace line")
	app.Trace("j.test.trace", "A fourth trace line")
	app.Warning("j.test.warning", "A warning line")
	app.Warning("j.test.warning", "Another warning line")
	app.Warning("j.test.warning", "Yet another warning line")
	app.Error("j.test.error", "An error line")
	app.Error("j.test.error", "Another error line")

	tests := []struct {
		level    log.Level
		expected int
	}{
		{level: log.LevelTrace, expected: 4},
		{level: log.LevelWarning, expected: 3},
		{level: log.LevelError, expected: 2},
	}

	// Check number of points
	expectTot := 0
	for _, c := range tests {
		expectTot += c.expected
	}
	gotTot := len(stats.Data[key])
	if expectTot != gotTot {
		tt.Fatalf("expect %s to get %d stats, but got %d", key, expectTot, gotTot)
	}

	// Get each points
	got := map[string]int{}
	for _, point := range stats.Data[key] {
		if point.Op != lt.OpHistogram {
			tt.Fatalf("expect to op %d, but got %d", lt.OpHistogram, point.Op)
		}

		if len(point.Meta) < 1 {
			tt.Fatal("expect to get at least one meta map")
		}

		lvl := point.Meta[0]["level"]
		got[lvl]++
	}

	// Check each point
	for _, test := range tests {
		got := got[test.level.String()]
		if got != test.expected {
			tt.Errorf("expect %s to have been called %d times, but got %d",
				test.level.String(),
				test.expected,
				got)
		}
	}
}

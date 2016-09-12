package app_test

import (
	"testing"

	lt "github.com/stairlin/lego/testing"
)

func TestLogger(t *testing.T) {
	tt := lt.New(t)
	app := tt.NewAppCtx("journey-test")

	// Send a few log lines
	app.Trace("j.test.trace", "A trace line")
	app.Tracef("j.test.trace", "A %s trace line", "formatted")
	app.Tracef("j.test.trace2", "Another %s trace line", "formatted")
	app.Tracef("j.test.trace3", "Yet another %s trace line", "formatted")
	app.Warning("A warning line")
	app.Warning("Another warning line")
	app.Warningf("A %s warning line", "formatted")
	app.Error("A error line")
	app.Errorf("A %s error line", "formatted")

	logger := tt.Logger().(*lt.Logger)
	tests := []struct {
		severity string
		expected int
	}{
		{severity: lt.TC, expected: 4},
		{severity: lt.WN, expected: 3},
		{severity: lt.ER, expected: 2},
	}

	// Ensure they have been sent to the logger
	for _, test := range tests {
		res := logger.Lines(test.severity)
		if res != test.expected {
			tt.Errorf(
				"expect logger to receive %d log lines for severity <%s>, but got %d",
				test.expected,
				test.severity,
				res,
			)
		}
	}
}

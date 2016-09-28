package app_test

import (
	"testing"

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

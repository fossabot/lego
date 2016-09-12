package app_test

import (
	"testing"

	lt "github.com/stairlin/lego/testing"
)

func TestLogger(t *testing.T) {
	tt := lt.New(t)
	app := tt.NewAppCtx("journey-test")

	// Send a few log lines
	app.Debug("j.test.debug", "A debug line")
	app.Debugf("j.test.debug", "A %s debug line", "formatted")
	app.Info("j.test.info", "A info line")
	app.Infof("j.test.info", "A %s info line", "formatted")
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
		{severity: lt.I, expected: 4},
		{severity: lt.W, expected: 3},
		{severity: lt.E, expected: 2},
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

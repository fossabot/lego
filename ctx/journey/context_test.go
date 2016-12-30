package journey_test

import (
	"strings"
	"testing"
	"time"

	netCtx "golang.org/x/net/context"

	"github.com/stairlin/lego/ctx/journey"
	lt "github.com/stairlin/lego/testing"
)

// TestUniqueness tests whether the context ID is unique
func TestUniqueness(t *testing.T) {
	tt := lt.New(t)
	j := journey.New(tt.NewAppCtx("journey-test"))
	other := journey.New(tt.NewAppCtx("journey-test"))

	if j.UUID() == other.UUID() {
		tt.Error("expect context to have different UUIDs")
	}
}

// TestShortID tests whether ShortID is a substring of ID
func TestShortID(t *testing.T) {
	tt := lt.New(t)
	j := journey.New(tt.NewAppCtx("journey-test"))

	if !strings.HasPrefix(j.UUID(), j.ShortID()) {
		tt.Error("expect ShortID to be a substring of UUID", j.UUID(), j.ShortID())
	}
}

// TestAppConfig tests whether it returns the given app config
func TestAppConfig(t *testing.T) {
	tt := lt.New(t)
	app := tt.NewAppCtx("journey-test")
	j := journey.New(app)

	if j.AppConfig() != app.Config() {
		tt.Errorf("expect AppConfig to return the given app config (%v - %v)", j.AppConfig(), app.Config())
	}
}

func TestLogger(t *testing.T) {
	tt := lt.New(t)
	app := tt.NewAppCtx("journey-test")
	j := journey.New(app)
	logger := app.L().(*lt.Logger)

	// Send a few log lines
	j.Trace("j.test.trace", "A trace line")
	j.Trace("j.test.trace", "A second trace line")
	j.Trace("j.test.trace", "A third trace line")
	j.Trace("j.test.trace", "A fourth trace line")
	j.Warning("j.test.warning", "A warning line")
	j.Warning("j.test.warning", "Another warning line")
	j.Warning("j.test.warning", "Yet another warning line")
	j.Error("j.test.error", "An error line")
	j.Error("j.test.error", "Another error line")

	tests := []struct {
		severity string
		expected int
	}{
		{severity: lt.TC, expected: 4 + 1},
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

// TestCancellation ensures that the context is being released upon cancellation
func TestCancellation(t *testing.T) {
	tt := lt.New(t)
	app := tt.NewAppCtx("journey-test")
	j := journey.New(app)

	j.Cancel()

	select {
	case <-j.Done():
		tt.Log("cancel released the context")
		expect := netCtx.Canceled
		if j.Err() != expect {
			tt.Errorf("expect error to be <%s>, but got <%s>", expect, j.Err())
		}
	case <-time.After(time.Microsecond * 250):
		tt.Error("expect cancel to release the context")
	}
}

// TestTimeout ensures that the context is being release after the given timeout
func TestTimeout(t *testing.T) {
	tt := lt.New(t)
	app := tt.NewAppCtx("journey-test")

	app.Config().Request.TimeoutMS = 1

	j := journey.New(app)

	select {
	case <-j.Done():
		tt.Log("timeout released the context")
		expect := netCtx.DeadlineExceeded
		if j.Err() != expect {
			tt.Errorf("expect error to be <%s>, but got <%s>", expect, j.Err())
		}
	case <-time.After(time.Millisecond * (app.Config().Request.TimeoutMS + 50)):
		tt.Error("expect cancel to release the context")
	}
}

// TestEnd ensures that the context is being release without errors when End() is called
func TestEnd(t *testing.T) {
	tt := lt.New(t)
	app := tt.NewAppCtx("journey-test")
	j := journey.New(app)

	j.End()

	select {
	case <-j.Done():
		tt.Log("end released the context")
		expect := netCtx.Canceled
		if j.Err() != expect {
			tt.Errorf("expect error to be <%s>, but got <%s>", expect, j.Err())
		}
	case <-time.After(time.Microsecond * 250):
		tt.Error("expect cancel to release the context")
	}
}

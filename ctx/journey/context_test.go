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
	tt.DisableStrictMode()

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

// TestCancellationPropagation ensures that the cancellation propagates
// to sub contexts when the root context is being cancelled
func TestCancellationPropagation(t *testing.T) {
	tt := lt.New(t)
	app := tt.NewAppCtx("journey-test")
	root := journey.New(app)
	child := root.BranchOff(journey.Child)
	grandchild := child.BranchOff(journey.Child)

	root.Cancel()

	for i, ctx := range []journey.Ctx{root, child, grandchild} {
		select {
		case <-ctx.Done():
			tt.Log("cancel released the context")
			expect := netCtx.Canceled
			if ctx.Err() != expect {
				tt.Errorf("%d - expect error to be <%s>, but got <%s>", i, expect, ctx.Err())
			}
		case <-time.After(time.Microsecond * 250):
			tt.Errorf("%d - expect cancel to release the context", i)
		}
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

// TestBG_Context ensures that the given context is a child context
func TestBG_Context(t *testing.T) {
	tt := lt.New(t)
	app := tt.NewAppCtx("journey-test")
	j := journey.New(app)

	res := make(chan journey.Ctx, 1)
	j.BG(func(c journey.Ctx) {
		res <- c
	})

	select {
	case c := <-res:
		if c == j {
			tt.Error("expect BG context to be different than parent context")
		}
	case <-time.After(time.Microsecond * 250):
		tt.Error("expect to receive context, but got nothing")
	}
}

// TestBG_Cancellation ensures that the parent context cannot cancel a background context
func TestBG_Cancellation(t *testing.T) {
	tt := lt.New(t)
	app := tt.NewAppCtx("journey-test")
	root := journey.New(app)

	res := make(chan journey.Ctx, 1)
	root.BG(func(c journey.Ctx) {
		res <- c
		time.Sleep(time.Millisecond * 5)
	})

	var child journey.Ctx
	select {
	case c := <-res:
		child = c
	}

	root.Cancel()

	select {
	case <-child.Done():
		tt.Error("expect to child context to be running")
	default:
		tt.Log("child context still running")
	}
}

func TestJourney_Marshalling(t *testing.T) {
	tt := lt.New(t)
	app := tt.NewAppCtx("journey-test")
	ctx := journey.New(app)

	// Create an encoder and send a value.
	data, err := journey.MarshalGob(ctx)
	if err != nil {
		t.Fatal("Marshal:", err)
	}

	// Decode value
	ctx, err = journey.UnmarshalGob(app, data)
	if err != nil {
		t.Fatal("Unmarshal:", err)
	}
}

func TestJourney_MarshallingKV(t *testing.T) {
	tt := lt.New(t)
	app := tt.NewAppCtx("journey-test")
	ctx := journey.New(app)

	ctx.KV().Store("foo", "bar")

	// Create an encoder and send a value.
	data, err := journey.MarshalGob(ctx)
	if err != nil {
		t.Fatal("Marshal:", err)
	}

	// Decode value
	ctx, err = journey.UnmarshalGob(app, data)
	if err != nil {
		t.Fatal("Unmarshal:", err)
	}

	v, ok := ctx.KV().Load("foo")
	if !ok {
		t.Fatal("expect kv to have key foo")
	}
	if v != "bar" {
		t.Fatalf("expect value bar, but got %s", v)
	}
}

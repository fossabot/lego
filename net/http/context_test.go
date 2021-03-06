package http_test

import (
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/stairlin/lego/ctx/app"
	"github.com/stairlin/lego/ctx/journey"
	"github.com/stairlin/lego/net/http"
	lt "github.com/stairlin/lego/testing"
)

type Foo struct {
	Label     string
	Threshold int
}

// TestDefaultBehaviour creates an HTTP endpoint and send a request from the client
// It ensures the context is NOT propagated upstream
func TestDefaultBehaviour(t *testing.T) {
	tt := lt.New(t)
	appCtx := tt.NewAppCtx("test-http")

	// Build handler
	h := http.NewServer()
	defer h.Drain()
	var gotContext journey.Ctx
	h.HandleFunc("/test", http.GET, func(
		ctx journey.Ctx, w http.ResponseWriter, r *http.Request,
	) {
		ctx.Trace("http.test", "Test endpoint called")
		gotContext = ctx
		w.Head(http.StatusOK)
	})

	addr := startServer(appCtx, h)

	// Prepare context
	ctx := journey.New(appCtx)
	ctx.Trace("prepare", "Prepare context")
	ctx.Store("lang", "en_GB")
	ctx.Store("ip", "10.0.0.21")
	ctx.Store("flag", 3)

	// Send request
	client := http.Client{}
	res, err := client.Get(ctx, fmt.Sprintf("http://%s/test", addr))
	if err != nil {
		t.Fatal(err)
	}
	if http.StatusOK != res.StatusCode {
		t.Errorf("expect to get status %d, but got %d", http.StatusOK, res.StatusCode)
	}

	// Compare
	if ctx.UUID() == gotContext.UUID() {
		t.Error("expect contexts to be different")
	}
	ctx.RangeValues(func(key, expect interface{}) bool {
		v := gotContext.Load(key)
		if v != nil {
			t.Errorf("expect key %s to NOT be present", key)
		}
		return false
	})
}

// TestAllowContext creates an HTTP endpoint and send a request from the client
// It ensures the context is NOT propagated upstream by the client by default
func TestAllowContext(t *testing.T) {
	tt := lt.New(t)
	appCtx := tt.NewAppCtx("test-http")
	appCtx.Config().Request.AllowContext = true

	// Build handler
	h := http.NewServer()
	var gotContext journey.Ctx
	h.HandleFunc("/test", http.GET, func(
		ctx journey.Ctx, w http.ResponseWriter, r *http.Request,
	) {
		ctx.Trace("http.test", "Test endpoint called")
		gotContext = ctx
		w.Head(http.StatusOK)
	})

	addr := startServer(appCtx, h)

	// Prepare context
	ctx := journey.New(appCtx)
	ctx.Trace("prepare", "Prepare context")
	ctx.Store("lang", "en_GB")
	ctx.Store("ip", "10.0.0.21")
	ctx.Store("flag", 3)

	// Send request
	client := http.Client{}
	res, err := client.Get(ctx, fmt.Sprintf("http://%s/test", addr))
	if err != nil {
		t.Fatal(err)
	}
	if http.StatusOK != res.StatusCode {
		t.Errorf("expect to get status %d, but got %d", http.StatusOK, res.StatusCode)
	}

	// Compare
	if ctx.UUID() == gotContext.UUID() {
		t.Error("expect contexts to be different")
	}
	ctx.RangeValues(func(key, expect interface{}) bool {
		v := gotContext.Load(key)
		if v != nil {
			t.Errorf("expect key %s to NOT be present", key)
		}
		return false
	})
}

// TestAllowContext creates an HTTP endpoint and send a request from the client
// It ensures the context is propagated, but blocked on the upstream node
func TestBlockContext(t *testing.T) {
	tt := lt.New(t)
	appCtx := tt.NewAppCtx("test-http")

	// Build handler
	h := http.NewServer()
	var gotContext journey.Ctx
	h.HandleFunc("/test", http.GET, func(
		ctx journey.Ctx, w http.ResponseWriter, r *http.Request,
	) {
		ctx.Trace("http.test", "Test endpoint called")
		gotContext = ctx
		w.Head(http.StatusOK)
	})

	addr := startServer(appCtx, h)

	// Prepare context
	ctx := journey.New(appCtx)
	ctx.Trace("prepare", "Prepare context")
	ctx.Store("lang", "en_GB")
	ctx.Store("ip", "10.0.0.21")
	ctx.Store("flag", 3)

	// Send request
	client := http.Client{
		PropagateContext: true,
	}
	res, err := client.Get(ctx, fmt.Sprintf("http://%s/test", addr))
	if err != nil {
		t.Fatal(err)
	}
	if http.StatusOK != res.StatusCode {
		t.Errorf("expect to get status %d, but got %d", http.StatusOK, res.StatusCode)
	}

	// Compare
	if ctx.UUID() == gotContext.UUID() {
		t.Error("expect contexts to be different")
	}
	ctx.RangeValues(func(key, expect interface{}) bool {
		v := gotContext.Load(key)
		if v != nil {
			t.Errorf("expect key %s to NOT be present", key)
		}
		return false
	})
}

// TestPropagateContext creates an HTTP endpoint and send a request from the client
// It ensures the context is propagated and accepted upstream
func TestPropagateContext(t *testing.T) {
	tt := lt.New(t)
	appCtx := tt.NewAppCtx("test-http")
	appCtx.Config().Request.AllowContext = true

	// Build handler
	h := http.NewServer()
	var gotContext journey.Ctx
	h.HandleFunc("/test", http.GET, func(
		ctx journey.Ctx, w http.ResponseWriter, r *http.Request,
	) {
		ctx.Trace("http.test", "Test endpoint called")
		gotContext = ctx
		w.Head(http.StatusOK)
	})

	addr := startServer(appCtx, h)

	// Prepare context
	ctx := journey.New(appCtx)
	ctx.Trace("prepare", "Prepare context")
	ctx.Store("lang", "en_GB")
	ctx.Store("ip", "10.0.0.21")
	ctx.Store("flag", 3)

	// Send request
	client := http.Client{
		PropagateContext: true,
	}
	res, err := client.Get(ctx, fmt.Sprintf("http://%s/test", addr))
	if err != nil {
		t.Fatal(err)
	}
	if http.StatusOK != res.StatusCode {
		t.Errorf("expect to get status %d, but got %d", http.StatusOK, res.StatusCode)
	}

	// Compare
	if ctx.UUID() != gotContext.UUID() {
		t.Errorf("expect context to have UUID %s, but got %s", ctx.UUID(), gotContext.UUID())
	}
	if gotContext == nil {
		t.Fatalf("expect KV to not be nil")
	}
	ctx.RangeValues(func(key, expect interface{}) bool {
		got := gotContext.Load(key)
		if expect != got {
			t.Errorf("expect to value for key %s to be %v, but got %v", key, expect, got)
		}
		return false
	})
}

func startServer(appCtx app.Ctx, h *http.Server) string {
	addr := fmt.Sprintf("127.0.0.1:%d", lt.NextPort())
	h.HandleFunc("/preflight", http.GET, func(
		ctx journey.Ctx, w http.ResponseWriter, r *http.Request,
	) {
		w.Head(http.StatusOK)
	})

	// Start serving requests
	go func() {
		err := h.Serve(addr, appCtx)
		if err != nil {
			panic(err)
		}
	}()
	// Ensure HTTP handler is ready to serve requests
	for attempt := 1; attempt <= 10; attempt++ {
		ctx := journey.New(appCtx)
		res, err := http.Get(ctx, fmt.Sprintf("http://%s/preflight", addr))
		if err == nil && res.StatusCode == http.StatusOK {
			break
		}
		backoff := math.Pow(2, float64(attempt))
		time.Sleep(time.Millisecond * time.Duration(backoff))
	}

	return addr
}

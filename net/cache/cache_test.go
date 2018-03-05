package cache_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stairlin/lego/ctx/app"
	"github.com/stairlin/lego/net/cache"
	lt "github.com/stairlin/lego/testing"
)

func TestDrain(t *testing.T) {
	tt := lt.New(t)
	appCtx := tt.NewAppCtx("test-cache")

	h := cache.NewServer(appCtx.Cache())
	addr := startServer(appCtx, h)

	res, err := http.Get("http://" + addr)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != 400 {
		t.Errorf("expect status code 400, but got %d", res.StatusCode)
	}

	h.Drain()

	if _, err := http.Get("http://" + addr); err == nil {
		t.Error("expect to get an error, but got nothing")
	}
}

func startServer(appCtx app.Ctx, h *cache.Server) string {
	addr := fmt.Sprintf("127.0.0.1:%d", lt.NextPort())

	// Start serving requests
	go func() {
		err := h.Serve(addr, appCtx)
		if err != nil {
			panic(err)
		}
	}()
	time.Sleep(20 * time.Millisecond)

	return addr
}

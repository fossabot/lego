package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stairlin/lego/ctx/app"
	"github.com/stairlin/lego/ctx/journey"
	"google.golang.org/grpc"
)

func TestDrain(t *testing.T) {
	tt := lt.New(t)
	appCtx := tt.NewAppCtx("test-grpc")
	appCtx.Config().Request.AllowContext = true

	// Build server
	h := lgrpc.NewServer()
	h.RegisterService(&_Test_serviceDesc, &MyTestServer{
		appCtx: appCtx,
		t:      tt,
	})
	addr := startServer(appCtx, h)

	// Build client
	c, err := lgrpc.NewClient(appCtx, addr, grpc.WithInsecure())
	if err != nil {
		t.Fatal(err)
	}
	c.PropagateContext = true
	testClient := NewTestClient(c.GRPC)

	// Prepare context
	ctx := journey.New(appCtx)
	ctx.Trace("prepare", "Prepare context")
	ctx.Store("lang", "en_GB")
	ctx.Store("ip", "10.0.0.21")
	ctx.Store("flag", 3)

	// Start draining server
	h.Drain()

	_, err = testClient.Hello(ctx, &Request{Msg: "Ping"})
	if err == nil {
		t.Fatal("expect to get an error when the server is drained")
	}
	if !strings.Contains(err.Error(), "grpc: the connection is unavailable") &&
		!strings.Contains(err.Error(), "transport is closing") &&
		!strings.Contains(err.Error(), "all SubConns are in TransientFailure") {
		t.Errorf("unexpected error %s", err)
	}
}

type gRPCServer struct {
	appCtx app.Ctx
}

func (s *MyTestServer) Hello(
	context context.Context, req *Request,
) (*Response, error) {
	ctx, ok := context.(journey.Ctx)
	if !ok {
		return nil, errors.New("context is not a journey")
	}
	ctx.Trace("test.hello", "Calling Hello")

	lang, ok := ctx.Load("lang").(string)
	expectLang := "en_GB"
	if !ok || lang != "en_GB" {
		s.t.Errorf("expect lang %s, but got %s", expectLang, lang)
	}

	expectMsg := "Ping"
	if expectMsg != req.Msg {
		s.t.Errorf("expect to get %s, but got %s", expectMsg, req.Msg)
	}
	return &Response{Msg: "Pong"}, nil
}

func startServer(appCtx app.Ctx, h *lgrpc.Server) string {
	addr := fmt.Sprintf("127.0.0.1:%d", lt.NextPort())

	// Start serving requests
	go func() {
		err := h.Serve(addr, appCtx)
		if err != nil {
			panic(err)
		}
	}()
	time.Sleep(50 * time.Millisecond)

	return addr
}

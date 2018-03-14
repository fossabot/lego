package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/pkg/errors"
	"github.com/stairlin/lego"
	"github.com/stairlin/lego/ctx/app"
	"github.com/stairlin/lego/ctx/journey"
	"github.com/stairlin/lego/example/grpc/server/demo"
	"github.com/stairlin/lego/log"
	"github.com/stairlin/lego/net/grpc"
	context "golang.org/x/net/context"
)

func main() {
	err := start()
	if err != nil {
		fmt.Println("App error", err)
		os.Exit(1)
	}
}

type AppConfig struct {
	Foo string `json:"foo"`
}

func start() error {
	// Create lego
	config := &AppConfig{}
	app, err := lego.New("grpc-server", config)
	if err != nil {
		return errors.Wrap(err, "Problem initialising lego")
	}

	// Build gRPC server
	port, err := strconv.Atoi(os.Getenv("GRPC_PORT"))
	if err != nil {
		return errors.Wrap(err, "Problem parsing port")
	}
	s := grpc.NewServer()
	s.AppendUnaryMiddleware(traceMiddleware)

	// Register gRPC handler as a service
	err = app.RegisterService(&lego.ServiceRegistration{
		Name:   "grpc.demo",
		Host:   "127.0.0.1",
		Port:   uint16(port),
		Server: s,
	})
	if err != nil {
		return errors.Wrap(err, "Problem registering service")
	}

	// Register gRPC service
	demo.RegisterDemoServer(s.GRPC, &gRPCServer{
		node:   os.Getenv("NODE_NAME"),
		appCtx: app.Ctx(),
	})

	// Start serving requests
	err = app.Serve()
	if err != nil {
		return errors.Wrap(err, "Problem serving requests")
	}
	return nil
}

type gRPCServer struct {
	node   string
	appCtx app.Ctx
}

func (s *gRPCServer) Hello(
	context context.Context, req *demo.Request,
) (*demo.Response, error) {
	ctx, ok := context.(journey.Ctx)
	if !ok {
		return nil, errors.New("context is not a journey")
	}
	ctx.Trace("grpc.hello", "Calling Hello", log.String("node", s.node))

	return &demo.Response{Msg: s.node}, nil
}

func traceMiddleware(next grpc.UnaryHandler) grpc.UnaryHandler {
	return func(ctx journey.Ctx, req interface{}) (interface{}, error) {
		ctx.Trace("grpc.trace.start", "Start call")
		res, err := next(ctx, req)
		ctx.Trace("grpc.trace.end", "End call")
		return res, err
	}
}

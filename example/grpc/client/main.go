package main

import (
	"fmt"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/stairlin/lego"
	"github.com/stairlin/lego/ctx/journey"
	"github.com/stairlin/lego/example/grpc/server/demo"
	lgrpc "github.com/stairlin/lego/net/grpc"
	"google.golang.org/grpc"
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
	app, err := lego.New("grpc-client", config)
	if err != nil {
		return errors.Wrap(err, "error initialising lego")
	}

	// Setup gRPC client
	c, err := lgrpc.NewClient(
		app.Ctx(),
		"passthrough://127.0.0.1:"+os.Getenv("GRPC_PORT"),
		grpc.WithInsecure(),
		grpc.WithTimeout(time.Second*60),
	)
	if err != nil {
		return errors.Wrap(err, "error connecting to server")
	}
	c.PropagateContext = true

	// Setup demo service
	demoSvc := demo.NewDemoClient(c.GRPC)

	// Prepare context
	ctx := journey.New(app.Ctx())
	ctx.Trace("prepare", "Prepare context")
	ctx.Store("lang", "en_GB")
	ctx.Store("ip", "10.0.0.21")
	ctx.Store("flag", 3)

	// Call service
	res, err := demoSvc.Hello(ctx, &demo.Request{Msg: "Ping"})
	if err != nil {
		return errors.Wrap(err, "grpc call failed")
	}

	fmt.Println("Hello service returned", res.Msg)
	return nil
}

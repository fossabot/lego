// Package main is a distributed-cache example
//
// It creates a cache server and register it to the service discovery agent
package main

import (
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/stairlin/lego"
	"github.com/stairlin/lego/ctx/journey"
	"github.com/stairlin/lego/net/http"
)

type AppConfig struct {
	Foo string `json:"foo"`
}

var (
	seededRand = rand.New(rand.NewSource(time.Now().UnixNano()))
	charset    = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
)

func main() {
	// Create lego
	config := &AppConfig{}
	app, err := lego.New("grpc", config)
	if err != nil {
		fmt.Println("Problem initialising lego", err)
		os.Exit(1)
	}

	port, err := strconv.Atoi(os.Getenv("HTTP_PORT"))
	if err != nil {
		fmt.Println("Problem parsing port", err)
		os.Exit(1)
	}

	hs := http.NewServer()

	api := &publicAPI{}
	hs.HandleFunc("hello", http.GET, api.Hello)

	err = app.RegisterService(&lego.ServiceRegistration{
		Name:   "api.cache",
		Host:   "127.0.0.1",
		Port:   uint16(port),
		Server: hs,
	})
	if err != nil {
		fmt.Println("Problem registering service", err)
		os.Exit(1)
	}

	// Start serving requests
	err = app.Serve()
	if err != nil {
		fmt.Println("Problem serving requests", err)
		os.Exit(1)
	}
}

type publicAPI struct{}

func (h *publicAPI) Hello(ctx journey.Ctx, w http.ResponseWriter, r *http.Request) {
	ctx.Trace("http.hello", "Hello")
}

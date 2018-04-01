// Package main is a distributed-cache example
//
// It creates a cache server and register it to the service discovery agent
package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/pkg/errors"
	"github.com/stairlin/lego"
	"github.com/stairlin/lego/net/schedule"
)

type AppConfig struct {
	Foo string `json:"foo"`
}

func main() {
	// Create lego
	config := &AppConfig{}
	app, err := lego.New("api", config)
	if err != nil {
		fmt.Println("error initialising lego app", err)
		os.Exit(1)
	}

	if err := start(app); err != nil {
		fmt.Println("start returned an error", err)
	}
}

func start(app *lego.App) error {
	port, err := strconv.Atoi(os.Getenv("SCHEDULE_PORT"))
	if err != nil {
		return errors.Wrap(err, "error parsing port")
	}
	tags := []string{"v1"}

	// Create schedule server
	server := schedule.NewServer()

	// Register it as a service
	err = app.RegisterService(&lego.ServiceRegistration{
		Name:   "schedule.local",
		Host:   "127.0.0.1",
		Port:   uint16(port),
		Server: server,
		Tags:   tags,
	})
	if err != nil {
		return errors.Wrap(err, "error registering service")
	}

	svc, err := app.Disco().Service(app.Ctx(), "schedule.local", "v1")
	if err != nil {
		return errors.Wrap(err, "error getting service")
	}
	server.AddOptions(schedule.OptDisco(svc))

	// Start serving requests
	err = app.Serve()
	if err != nil {
		return errors.Wrap(err, "error serving requests")
	}
	return nil
}

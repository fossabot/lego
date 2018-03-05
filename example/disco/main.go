// Package main is a service discovery example
//
// It creates an HTTP server and register it to the service discovery agent
package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"

	"github.com/stairlin/lego"
	"github.com/stairlin/lego/ctx/journey"
	"github.com/stairlin/lego/log"
	"github.com/stairlin/lego/net/http"
)

type AppConfig struct {
	Foo string `json:"foo"`
}

func main() {
	// Create lego
	config := &AppConfig{}
	app, err := lego.New("disco", config)
	if err != nil {
		fmt.Println("Problem initialising lego", err)
		os.Exit(1)
	}

	port, err := strconv.Atoi(os.Getenv("HTTP_PORT"))
	if err != nil {
		fmt.Println("Problem parsing port", err)
		os.Exit(1)
	}
	tags := []string{"api", "http", app.Config().Version}

	// Register service
	s := http.NewServer()
	s.HandleFunc("/hello", http.GET, Hello)
	err = app.RegisterService(&lego.ServiceRegistration{
		Name:   "api.http",
		Host:   "127.0.0.1",
		Port:   uint16(port),
		Server: s,
		Tags:   tags,
	})
	if err != nil {
		fmt.Println("Problem registering service", err)
		os.Exit(1)
	}

	// Listen to service discovery events for that service
	svc, err := app.Disco().Service(app.Ctx(), "api.http", tags...)
	if err != nil {
		fmt.Println("Problem getting service", err)
		os.Exit(1)
	}
	sub, unsub := svc.Sub()
	go func() {
		for {
			select {
			case e := <-sub:
				app.Ctx().Trace("example.disco.event", "Got event",
					log.Int("instances", len(e.Instances)),
				)
			}
		}
	}()
	defer unsub()

	// Start serving requests
	err = app.Serve()
	if err != nil {
		fmt.Println("Problem serving requests", err)
		os.Exit(1)
	}
}

// Hello handler example
func Hello(ctx journey.Ctx, w http.ResponseWriter, r *http.Request) {
	ctx.Trace("http.hello", "Hello called")
	text := "Hello from " + ctx.AppConfig().Node
	w.Data(
		http.StatusOK,
		"text/plain",
		ioutil.NopCloser(bytes.NewReader([]byte(text))),
	)
}

package main

import (
	"fmt"
	"os"

	"github.com/stairlin/lego"
	"github.com/stairlin/lego/net/http"
	"github.com/stairlin/lego/log"
)

type AppConfig struct {
	Foo string `json:"foo"`
}

func main() {
	// Create lego
	config := &AppConfig{}
	app, err := lego.New("api", config)
	if err != nil {
		fmt.Println("Problem initialising lego", err)
		os.Exit(1)
	}

	// Register HTTP handler
	h := http.NewHandler()
	h.HandleFunc("/ping", http.GET, Ping)
	app.RegisterHandler("127.0.0.1:3000", h)

	// Start serving requests
	err = app.Serve()
	if err != nil {
		fmt.Println("Problem serving requests", err)
		os.Exit(1)
	}
}

// Ping handler example
func Ping(c *http.Context) http.Renderer {
	c.Ctx.Trace("action.ping", "Simple request", log.Time("start_time", c.StartTime))
	return c.Head(http.StatusOK)
}

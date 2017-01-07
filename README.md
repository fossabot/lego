# lego [![CircleCI](https://circleci.com/gh/stairlin/lego.svg?style=svg)](https://circleci.com/gh/stairlin/lego) [![Go Report Card](https://goreportcard.com/badge/github.com/stairlin/lego)](https://goreportcard.com/report/github.com/stairlin/lego)

```shell
CONFIG_URI=file://${PWD}/config.json go run main.go
```

## Setup

Basic setup

```go
package main

import (
	"fmt"
	"os"

	"github.com/stairlin/lego"
	"github.com/stairlin/lego/handler/http"
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
	h.Handle("/ping", http.GET, &Ping{})
	app.RegisterHandler("127.0.0.1:3000", h)

	// Start serving requests
	err = app.Serve()
	if err != nil {
		fmt.Println("Problem serving requests", err)
		os.Exit(1)
	}
}

// HTTP handler example
type Ping struct{}

func (a *Ping) Call(c *http.Context) http.Renderer {
	c.Ctx.Trace("action.ping", "Simple request", log.Time("start_at", c.StartAt))
	return c.Head(http.StatusOK)
}

```

## Config

Example of a configuration file

```json
{
    "node": "node.test",
    "version": "test",
    "log": {
        "level": "trace",
        "formatter": {
            "adapter": "logf"
        },
        "printer": {
            "adapter": "stdout"
        }
    },
    "stats": {
        "on": false
    },
    "request": {
        "timeout_ms": 500
    },
    "app": {
        "foo": "bar"
    }
}
```

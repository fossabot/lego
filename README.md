# lego [![CircleCI](https://circleci.com/gh/stairlin/lego.svg?style=svg)](https://circleci.com/gh/stairlin/lego)

```shell
CONFIG_URI=file://${PWD}/config/dev.json go run main.go -logtostderr
```

## Setup

Basic setup

```go
package main

import (
	"flag"
	"fmt"
	"os"

    "github.com/stairlin/lego"
    "github.com/stairlin/lego/handler/http"
)

func main() {
    flag.Parse()

    // Create lego
    app, err := lego.New("api", nil)
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
    c.Ctx.Info("action.ping")
    return c.Head(http.StatusOK)
}
```

## Config

Example of a configuration file

```json
{
  "log": {
    "level": "trace",
    "output": "console",
    "config": { }
  },
  "stats": {
    "on": false
  }
}
```

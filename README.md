# lego

```go
    package main

    import (
        "github.com/stairlin/lego"
        "github.com/stairlin/lego/handler/http"
    )

    func main() {
        // Create lego
        app, err := lego.New("api", nil)
        if err != nil {
            fmt.Println("Problem initialising lego", err)
            os.Exit(1)
        }

        // Register HTTP handler
        h := http.NewHandler()
        h.Handle("/ping", http.GET, &Ping{})
        app.RegisterHandler(":3000", h)

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
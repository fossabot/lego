package lego

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/stairlin/lego/config"
	"github.com/stairlin/lego/ctx/app"
	"github.com/stairlin/lego/handler"
	"github.com/stairlin/lego/log"
	statsAdapter "github.com/stairlin/lego/stats/adapter"
)

// App is the core structure for a new service
type App struct {
	mu    sync.Mutex
	ready *sync.Cond

	ctx      app.Ctx
	config   *config.Config
	handlers *handler.Reg
	drain    bool
	done     chan bool
}

// New creates a new App and returns it
func New(name string, appConfig interface{}) (*App, error) {
	// Get config store
	configStore, err := config.NewStore(os.Getenv("CONFIG_URI"))
	if err != nil {
		return nil, fmt.Errorf("config store error: %s", err)
	}

	// Load config from store
	c := &config.Config{App: appConfig}
	err = configStore.Load(c)
	if err != nil {
		return nil, fmt.Errorf("cannot load config: %s", err)
	}

	// Create logger
	// TODO : Extract to log package (per adapter)
	l, err := log.NewStdLogger(c.Log.Level)
	if err != nil {
		return nil, fmt.Errorf("logger error: %s", err)
	}

	// Build stats
	s, err := statsAdapter.New(&c.Stats)
	if err != nil {
		return nil, fmt.Errorf("stats error: %s", err)
	}
	s.SetLogger(l)

	// Build app context
	ctx := app.NewCtx(name, c, l, s)

	// Build ready cond flag
	lock := &sync.Mutex{}
	lock.Lock()
	ready := sync.NewCond(lock)

	// Build app struct
	app := &App{
		ready:    ready,
		ctx:      ctx,
		handlers: handler.NewReg(ctx),
		done:     make(chan bool, 1),
	}

	// Start background services
	ctx.BG().Dispatch(s)
	ctx.BG().Dispatch(&hearbeat{app: app})

	// Trap OS signals
	go trapSignals(app)

	return app, nil
}

// Config returns the lego config
func (a *App) Config() *config.Config {
	return a.config
}

// Serve allows handlers to serve requests and blocks the call
func (a *App) Serve() error {
	a.ctx.Tracef("lego.serve", "start serving...")

	err := a.handlers.Serve()
	if err != nil {
		a.ctx.Errorf("Error with handler.Serve (%s)", err)
		return err
	}

	// Notify all callees that the app is up and running
	a.ready.Broadcast()

	<-a.done // Hang on
	return nil
}

// Ready holds the callee until the app is fully operational
func (a *App) Ready() {
	a.ready.Wait()
}

// Drain notify all handlers to enter in draining mode. It means they are no
// longer accepting new requests, but they can finish all in-flight requests
func (a *App) Drain() {
	a.ctx.Tracef("lego.drain", "start draining...")

	// Check if we are already stopping
	a.mu.Lock()
	if a.drain {
		a.mu.Unlock()
		return
	}
	a.drain = true
	a.mu.Unlock()

	a.handlers.Drain() // Block all new requests and drain in-flight requests
	a.ctx.BG().Drain() // Now drain last background services

	a.done <- true // Release Serve()
}

// Ctx returns the appliation context
func (a *App) Ctx() app.Ctx {
	return a.ctx
}

func trapSignals(app *App) {
	ch := make(chan os.Signal, 10)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGKILL, syscall.SIGTERM)

	for {
		sig := <-ch
		app.Ctx().Tracef("lego.signal", "trapped <%s>", sig)

		switch sig {
		case syscall.SIGINT, syscall.SIGTERM:
			app.Drain() // start draining handlers
			signal.Stop(ch)
			return
		case syscall.SIGKILL:
			signal.Stop(ch)
			return
		}
	}
}

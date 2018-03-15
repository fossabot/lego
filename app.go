package lego

import (
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"
	"strconv"
	"sync"
	"syscall"

	"github.com/pkg/errors"
	"github.com/stairlin/lego/config"
	"github.com/stairlin/lego/ctx/app"
	"github.com/stairlin/lego/disco"
	da "github.com/stairlin/lego/disco/adapter"
	"github.com/stairlin/lego/log"
	"github.com/stairlin/lego/log/logger"
	"github.com/stairlin/lego/net"
	sa "github.com/stairlin/lego/stats/adapter"
)

// App is the core structure for a new service
type App struct {
	mu    sync.Mutex
	ready *sync.Cond

	service string
	ctx     app.Ctx
	config  *config.Config
	disco   disco.Agent
	servers *net.Reg
	drain   bool
	done    chan bool
}

// New creates a new App and returns it
func New(service string, appConfig interface{}) (*App, error) {
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

	// Convert potential environment variables
	c.Node = config.ValueOf(c.Node)
	c.Version = config.ValueOf(c.Version)

	return NewWithConfig(service, c)
}

// NewWithConfig creates a new App with the config config
func NewWithConfig(service string, c *config.Config) (*App, error) {
	// Create logger
	l, err := logger.New(service, &c.Log)
	if err != nil {
		return nil, fmt.Errorf("logger error: %s", err)
	}

	// Build stats
	s, err := sa.New(&c.Stats)
	if err != nil {
		return nil, fmt.Errorf("stats error: %s", err)
	}
	s.SetLogger(l)

	// Service discovery
	sd, err := da.New(c)
	if err != nil {
		return nil, fmt.Errorf("disco error: %s", err)
	}

	// Build app context
	ctx := app.NewCtx(service, c, l, s, sd)

	// Build ready cond flag
	lock := &sync.Mutex{}
	lock.Lock()
	ready := sync.NewCond(lock)

	// Build app struct
	app := &App{
		service: service,
		ready:   ready,
		ctx:     ctx,
		config:  c,
		disco:   sd,
		servers: net.NewReg(ctx),
		done:    make(chan bool, 1),
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
	defer func() {
		if recover := recover(); recover != nil {
			a.Ctx().Error("lego.serve.panic", "App panic",
				log.Object("err", recover),
				log.String("stack", string(debug.Stack())),
			)

			// Attempt to clean resources before propagating the panic further up
			if !a.drain {
				a.Drain()
			}

			panic(recover)
		}
	}()

	a.ctx.Trace("lego.serve", "Start serving...")

	err := a.servers.Serve()
	if err != nil {
		a.ctx.Error("lego.serve.error", "Error with handler.Serve (%s)",
			log.Error(err),
		)
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
	a.ctx.Trace("lego.drain", "Start draining...")

	// Check if we are already stopping
	a.mu.Lock()
	if a.drain {
		a.mu.Unlock()
		return
	}
	a.drain = true
	a.mu.Unlock()

	a.servers.Drain() // Block all new requests and drain in-flight requests
	a.ctx.Drain()

	a.disco.Leave(a.ctx)
	a.ctx.Cancel()

	a.done <- true // Release Serve()
}

// Ctx returns the appliation context
func (a *App) Ctx() app.Ctx {
	return a.ctx
}

// RegisterServer adds the given server to the list of managed servers
func (a *App) RegisterServer(addr string, s net.Server) {
	a.servers.Add(addr, s)
}

// ServiceRegistration contains info to register a service
type ServiceRegistration struct {
	Name   string
	Host   string
	Port   uint16
	Server net.Server
	Tags   []string
}

// RegisterService adds the server to the list of managed servers and registers
// it to service discovery
func (a *App) RegisterService(r *ServiceRegistration) error {
	a.servers.Add(net.JoinHostPort(r.Host, strconv.Itoa(int(r.Port))), r.Server)

	dr := disco.Registration{
		Name: r.Name,
		Addr: r.Host,
		Port: r.Port,
		Tags: append(r.Tags, a.service, a.config.Version),
	}
	if _, err := a.disco.Register(a.Ctx(), &dr); err != nil {
		return errors.Wrap(err, "error registering service")
	}
	return nil
}

// Disco returns the active service discovery agent.
//
// When service discovery is disabled, it will return a local agent that acts
// like a regular service discovery agent, expect that it only registers local
// services.
func (a *App) Disco() disco.Agent {
	return a.disco
}

func trapSignals(app *App) {
	ch := make(chan os.Signal, 10)
	signals := []os.Signal{
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGQUIT,
		syscall.SIGABRT,
		syscall.SIGKILL,
		syscall.SIGTERM,
		syscall.SIGUSR1,
		syscall.SIGUSR2,
	}
	signal.Notify(ch, signals...)

	for {
		sig := <-ch
		app.Ctx().Trace("lego.signal", "Signal trapped", log.String("sig", sig.String()))

		switch sig {
		case syscall.SIGINT, syscall.SIGTERM:
			app.Drain() // start draining handlers
			signal.Stop(ch)
			return
		default:
			signal.Stop(ch)
			return
		}
	}
}

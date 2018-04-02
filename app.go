package lego

import (
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"
	"strconv"
	"sync"
	"sync/atomic"
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

const (
	down uint32 = iota
	up
	drain
)

// App is the core structure for a new service
type App struct {
	mu    sync.Mutex
	ready *sync.Cond

	service       string
	ctx           app.Ctx
	config        *config.Config
	disco         disco.Agent
	servers       *net.Reg
	registrations []*disco.Registration

	state uint32
	done  chan bool
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

			a.Close()
			panic(recover)
		}
	}()

	if !a.isState(down) {
		a.ctx.Warning("lego.serve.state", "Server is not in down state",
			log.Uint("state", uint(a.state)),
		)
	}

	a.ctx.Trace("lego.serve", "Start serving...")

	err := a.servers.Serve()
	if err != nil {
		a.ctx.Error("lego.serve.error", "Error with handler.Serve (%s)",
			log.Error(err),
		)
		return err
	}

	for _, reg := range a.registrations {
		_, err := a.disco.Register(a.Ctx(), reg)
		if err != nil {
			return errors.Wrapf(err, "error registering service <%s>", reg.Name)
		}
	}

	// Notify all callees that the app is up and running
	a.ready.Broadcast()

	atomic.StoreUint32(&a.state, up)
	<-a.done
	return nil
}

// Ready holds the callee until the app is fully operational
func (a *App) Ready() {
	a.ready.Wait()
}

// Drain notify all handlers to enter in draining mode. It means they are no
// longer accepting new requests, but they can finish all in-flight requests
func (a *App) Drain() bool {
	a.mu.Lock()
	if !a.isState(up) {
		a.mu.Unlock()
		return false
	}
	atomic.StoreUint32(&a.state, drain)
	a.mu.Unlock()

	a.ctx.Trace("lego.drain", "Start draining...")
	a.servers.Drain() // Block all new requests and drain in-flight requests
	a.ctx.Drain()
	return true
}

// Shutdown gracefully shuts down the server without interrupting any
// active connections. Shutdown works by first draining all handlers, then
// draining the main context, and finally shut down.
// If the provided context expires before the shutdown is complete,
// Shutdown returns the context's error, otherwise it returns any
// error returned from closing the Server's underlying Listener(s).
func (a *App) Shutdown() {
	a.ctx.Trace("lego.shutdown", "Gracefully shutting down...")
	a.disco.Leave(a.ctx)
	if !a.Drain() {
		a.ctx.Trace("lego.shutdown.abort", "Server already draining")
		return
	}
	a.close()
}

// Close immediately closes the server and any in-flight request or background
// job will be left unfinished.
// For a graceful shutdown, use Shutdown.
func (a *App) Close() error {
	a.ctx.Trace("lego.close", "Closing immediately!")
	a.disco.Leave(a.ctx)
	a.close()
	return nil
}

func (a *App) close() {
	a.ctx.Cancel()

	a.done <- true
	atomic.StoreUint32(&a.state, down)
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
	// ID is the service instance unique identifier (optional)
	ID string
	// Name is the service identifier
	Name string
	// Host is the interface on which the server runs.
	// Service discovery can override this value.
	Host string
	// Port is the port number
	Port uint16
	// Server is the server that provides the registered service
	Server net.Server
	// Tags for that service (versioning, blue-green, whatever)
	Tags []string
}

// RegisterService adds the server to the list of managed servers and registers
// it to service discovery
func (a *App) RegisterService(r *ServiceRegistration) {
	a.servers.Add(net.JoinHostPort(r.Host, strconv.Itoa(int(r.Port))), r.Server)

	a.registrations = append(a.registrations, &disco.Registration{
		ID:   r.ID,
		Name: r.Name,
		Addr: r.Host,
		Port: r.Port,
		Tags: append(r.Tags, a.service),
	})
}

// Disco returns the active service discovery agent.
//
// When service discovery is disabled, it will return a local agent that acts
// like a regular service discovery agent, expect that it only registers local
// services.
func (a *App) Disco() disco.Agent {
	return a.disco
}

// isState checks the current app state
func (a *App) isState(state uint32) bool {
	return atomic.LoadUint32(&a.state) == uint32(state)
}

func trapSignals(app *App) {
	ch := make(chan os.Signal, 10)
	signals := []os.Signal{
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
		syscall.SIGKILL,
	}
	signal.Notify(ch, signals...)

	for {
		sig := <-ch
		n, _ := sig.(syscall.Signal)
		app.Ctx().Trace("lego.signal", "Signal trapped",
			log.String("sig", sig.String()),
			log.Int("n", int(n)),
		)

		switch sig {
		case syscall.SIGINT, syscall.SIGTERM:
			app.Shutdown()
			signal.Stop(ch)
			return
		case syscall.SIGQUIT, syscall.SIGKILL:
			app.Close()
			signal.Stop(ch)
			return
		default:
			app.Ctx().Error("lego.signal.unhandled", "Unhandled signal")
			os.Exit(128 + int(n))
			return
		}
	}
}

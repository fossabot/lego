package lego

import (
	"context"
	"io"
	"os"
	"os/signal"
	"runtime/debug"
	"strconv"
	"sync"
	"sync/atomic"
	"syscall"

	"github.com/pkg/errors"
	"github.com/stairlin/lego/bg"
	"github.com/stairlin/lego/cache"
	"github.com/stairlin/lego/config"
	"github.com/stairlin/lego/ctx/app"
	"github.com/stairlin/lego/disco"
	discoA "github.com/stairlin/lego/disco/adapter"
	"github.com/stairlin/lego/log"
	"github.com/stairlin/lego/log/logger"
	"github.com/stairlin/lego/net"
	"github.com/stairlin/lego/schedule"
	scheduleA "github.com/stairlin/lego/schedule/adapter"
	"github.com/stairlin/lego/stats"
	statsA "github.com/stairlin/lego/stats/adapter"
)

const (
	down uint32 = iota
	up
	drain
)

// App is the core structure for a new service
type App struct {
	mu     sync.Mutex
	ready  *sync.Cond
	ctx    context.Context
	cancel context.CancelFunc

	service string
	config  config.Config
	state   uint32
	stopc   chan struct{}

	servers       *net.Reg
	registrations []*disco.Registration

	bg       *bg.Reg
	log      log.Logger
	stats    stats.Stats
	disco    disco.Agent
	cache    cache.Cache
	schedule schedule.Scheduler

	// TODO: Remove app.Ctx
	appCtx app.Ctx
}

// New creates a new App and returns it
func New(service string, appConfig interface{}) (*App, error) {
	configStore, err := config.NewStore(os.Getenv("CONFIG_URI"))
	if err != nil {
		return nil, errors.Wrap(err, "error creating config store")
	}

	r, err := configStore.Load()
	if err != nil {
		return nil, errors.Wrap(err, "error loading load config")
	}
	defer r.Close()

	return NewWithConfig(service, r, appConfig)
}

// NewWithConfig creates a new App with a custom configuration
func NewWithConfig(
	service string, r io.Reader, appConfig interface{},
) (a *App, err error) {
	configTree, err := config.LoadTree(r)
	if err != nil {
		return nil, errors.Wrap(err, "error loading config tree")
	}

	err = configTree.Get("app").Unmarshal(appConfig)
	if err != nil {
		return nil, errors.Wrap(err, "annot unmarshal app config")
	}

	// Build app struct
	lock := &sync.Mutex{}
	lock.Lock()
	ready := sync.NewCond(lock)
	ctx, cancelFunc := context.WithCancel(context.Background())
	a = &App{
		ready:   ready,
		ctx:     ctx,
		cancel:  cancelFunc,
		service: service,
		stopc:   make(chan struct{}),
	}

	err = configTree.Unmarshal(&a.config)
	if err != nil {
		return nil, errors.Wrap(err, "annot unmarshal core config")
	}

	// Set up services
	a.log, err = logger.New(service, configTree.Get("log"))
	if err != nil {
		return nil, errors.Wrap(err, "error initialising logger")
	}
	a.stats, err = statsA.New(configTree.Get("stats"))
	if err != nil {
		return nil, errors.Wrap(err, "error initialising stats")
	}
	a.disco, err = discoA.New(configTree.Get("disco"))
	if err != nil {
		return nil, errors.Wrap(err, "error initialising disco")
	}
	a.schedule, err = scheduleA.New(configTree.Get("schedule"))
	if err != nil {
		return nil, errors.Wrap(err, "error initialising scheduler")
	}
	a.cache = cache.New() // TODO: Cache adapter

	// Trap OS signals
	go trapSignals(a)

	// Create servers registry
	a.bg = bg.NewReg(service, a.log, a.stats)
	a.servers = net.NewReg(a.log)

	// Build app context
	a.appCtx = app.NewCtx(
		service,
		&a.config,
		a.log,
		a.stats,
		a.disco,
	)

	// Start background services
	a.BG().Dispatch(a.stats)
	a.BG().Dispatch(&hearbeat{app: a})

	if err := a.schedule.Start(a.appCtx); err != nil {
		return nil, errors.Wrap(err, "error starting scheduler")
	}
	return a, nil
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
		a.Warning("lego.serve.state", "Server is not in down state",
			log.Uint("state", uint(a.state)),
		)
	}

	a.Trace("lego.serve", "Start serving...")

	err := a.servers.Serve(a.appCtx)
	if err != nil {
		a.Error("lego.serve.error", "Error with handler.Serve (%s)",
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
	<-a.stopc
	return nil
}

// Ready holds the callee until the app is fully operational
func (a *App) Ready() {
	a.ready.Wait()
}

func (a *App) Service() string {
	return a.service
}

func (a *App) L() log.Logger {
	return a.log
}

func (a *App) Stats() stats.Stats {
	return a.stats
}

func (a *App) Config() *config.Config {
	return &a.config
}

func (a *App) BG() *bg.Reg {
	return a.bg
}

func (a *App) Cache() cache.Cache {
	return a.cache
}

// Disco returns the active service discovery agent.
//
// When service discovery is disabled, it will return a local agent that acts
// like a regular service discovery agent, expect that it only registers local
// services.
func (a *App) Disco() disco.Agent {
	return a.disco
}

func (a *App) Scheduler() schedule.Scheduler {
	return a.schedule
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

	a.Trace("lego.drain", "Start draining...")
	a.servers.Drain() // Block all new requests and drain in-flight requests
	a.appCtx.Drain()
	return true
}

// Shutdown gracefully shuts down the server without interrupting any
// active connections. Shutdown works by first draining all handlers, then
// draining the main context, and finally shut down.
// If the provided context expires before the shutdown is complete,
// Shutdown returns the context's error, otherwise it returns any
// error returned from closing the Server's underlying Listener(s).
func (a *App) Shutdown() {
	a.Trace("lego.shutdown", "Gracefully shutting down...")
	a.disco.Leave(a.appCtx)
	if !a.Drain() {
		a.Trace("lego.shutdown.abort", "Server already draining")
		return
	}
	a.close()
}

// Close immediately closes the server and any in-flight request or background
// job will be left unfinished.
// For a graceful shutdown, use Shutdown.
func (a *App) Close() error {
	a.Trace("lego.close", "Closing immediately!")
	a.disco.Leave(a.appCtx)
	a.close()
	return nil
}

func (a *App) close() {
	a.schedule.Close()
	a.appCtx.Cancel()

	select {
	case a.stopc <- struct{}{}:
	default:
	}
	atomic.StoreUint32(&a.state, down)
}

// Ctx returns the appliation context.
// DEPRECATED function. App will become a context
func (a *App) Ctx() app.Ctx {
	return a.appCtx
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

// isState checks the current app state
func (a *App) isState(state uint32) bool {
	return atomic.LoadUint32(&a.state) == uint32(state)
}

// Trace implements log.Logger
func (a *App) Trace(tag, msg string, fields ...log.Field) {
	a.log.Trace(tag, msg, fields...)
}

// Warning implements log.Logger
func (a *App) Warning(tag, msg string, fields ...log.Field) {
	a.log.Warning(tag, msg, fields...)
}

// Error implements log.Logger
func (a *App) Error(tag, msg string, fields ...log.Field) {
	a.log.Error(tag, msg, fields...)
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

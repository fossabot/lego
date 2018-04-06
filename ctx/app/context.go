// Package app defines an application context, which carries information about
// the application environment.
//
// It can be information such as database credentials, service discovery,
// handlers addresses, ...
package app

import (
	goc "context"
	"time"

	"github.com/stairlin/lego/bg"
	"github.com/stairlin/lego/cache"
	"github.com/stairlin/lego/config"
	"github.com/stairlin/lego/ctx"
	"github.com/stairlin/lego/disco"
	"github.com/stairlin/lego/log"
	"github.com/stairlin/lego/stats"
)

// Ctx is the app context interface
type Ctx interface {
	ctx.Ctx

	Service() string
	L() log.Logger
	Config() *config.Config
	BG() *bg.Reg
	Disco() disco.Agent
	Cache() cache.Cache
	Drain()
	Cancel()
}

// context holds the application context
type context struct {
	appConfig  *config.Config
	bgReg      *bg.Reg
	disco      disco.Agent
	cache      cache.Cache
	c          goc.Context
	cancelFunc goc.CancelFunc
	service    string
	l          log.Logger
	lFields    []log.Field
	stats      stats.Stats
}

// NewCtx creates a new app context
func NewCtx(
	service string,
	c *config.Config,
	l log.Logger,
	s stats.Stats,
	sd disco.Agent,
) Ctx {
	// Build background registry
	reg := bg.NewReg(service, l, s)

	lf := []log.Field{
		log.String("node", c.Node),
		log.String("version", c.Version),
		log.String("log_type", "A"),
	}

	ctx, cancelFunc := goc.WithCancel(goc.Background())

	return &context{
		service:    service,
		appConfig:  c,
		bgReg:      reg,
		disco:      sd,
		cache:      cache.New(),
		c:          ctx,
		cancelFunc: cancelFunc,
		l:          l.AddCalldepth(1),
		lFields:    lf,
		stats:      s,
	}
}

func (c *context) Service() string {
	return c.service
}

func (c *context) L() log.Logger {
	return c.l
}

func (c *context) Stats() stats.Stats {
	return c.stats
}

func (c *context) Config() *config.Config {
	return c.appConfig
}

func (c *context) BG() *bg.Reg {
	return c.bgReg
}

func (c *context) Disco() disco.Agent {
	return c.disco
}

func (c *context) Cache() cache.Cache {
	return c.cache
}

func (c *context) Drain() {
	c.bgReg.Drain()
}

func (c *context) Cancel() {
	c.cancelFunc()
}

// Trace level logs are to follow the code executio step by step
func (c *context) Trace(tag, msg string, fields ...log.Field) {
	c.l.Trace(tag, msg, c.logFields(fields)...)
	c.incLogLevelCount(log.LevelTrace, tag)
}

// Warning level logs are meant to draw attention above a certain threshold
func (c *context) Warning(tag, msg string, fields ...log.Field) {
	c.l.Warning(tag, msg, c.logFields(fields)...)
	c.incLogLevelCount(log.LevelWarning, tag)
}

// Error level logs need immediate attention
func (c *context) Error(tag, msg string, fields ...log.Field) {
	c.l.Error(tag, msg, c.logFields(fields)...)
	c.incLogLevelCount(log.LevelError, tag)
}

// # Net Context functions
// These are implemented in order to use a journey context as a net context

// Deadline returns the time when work done on behalf of this context
// should be canceled. Deadline returns ok==false when no deadline is
// set. Successive calls to Deadline return the same results.
func (c *context) Deadline() (deadline time.Time, ok bool) { return c.c.Deadline() }

// Done returns a channel that's closed when work done on behalf of this
// context should be canceled. Done may return nil if this context can
// never be canceled. Successive calls to Done return the same value.
func (c *context) Done() <-chan struct{} { return c.c.Done() }

// Err returns a non-nil error value after Done is closed. Err returns
// Canceled if the context was canceled or DeadlineExceeded if the
// context's deadline passed. No other values for Err are defined.
// After Done is closed, successive calls to Err return the same value.
func (c *context) Err() error { return c.c.Err() }

// Value returns the value associated with this context for key, or nil
// if no value is associated with key. Successive calls to Value with
// the same key returns the same result.
//
// Use context values only for request-scoped data that transits
// processes and API boundaries, not for passing optional parameters to
// functions.
func (c *context) Value(key interface{}) interface{} {
	c.Trace("ctx.app.value", "Add net context value", log.Object("value", key))
	return c.c.Value(key)
}

func (c *context) logFields(fields []log.Field) []log.Field {
	return append(c.lFields, fields...)
}

func (c *context) incLogLevelCount(lvl log.Level, tag string) {
	tags := map[string]string{
		"level":   lvl.String(),
		"tag":     tag,
		"service": c.service,
		"node":    c.appConfig.Node,
		"version": c.appConfig.Version,
	}

	c.stats.Histogram("log.level", 1, tags)
}

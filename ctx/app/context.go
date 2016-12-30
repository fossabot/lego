// Package app defines an application context, which carries information about
// the application environment.
//
// It can be information such as database credentials, service discovery,
// handlers addresses, ...
package app

import (
	netCtx "golang.org/x/net/context"

	"github.com/stairlin/lego/bg"
	"github.com/stairlin/lego/config"
	"github.com/stairlin/lego/ctx"
	"github.com/stairlin/lego/log"
	"github.com/stairlin/lego/stats"
)

// Ctx is the app context interface
type Ctx interface {
	ctx.Ctx

	L() log.Logger
	Config() *config.Config
	BG() *bg.Reg
	RootContext() netCtx.Context
}

// context holds the application context
type context struct {
	Service   string
	AppConfig *config.Config
	BGReg     *bg.Reg
	RootCtx   netCtx.Context

	l       log.Logger
	lFields []log.Field
	stats   stats.Stats
}

// NewCtx creates a new app context
func NewCtx(service string, c *config.Config, l log.Logger, s stats.Stats) Ctx {
	// Build background registry
	reg := bg.NewReg(service, l, s)

	lf := []log.Field{
		log.String("node", c.Node),
		log.String("version", c.Version),
		log.String("log_type", "A"),
	}

	return &context{
		Service:   service,
		AppConfig: c,
		BGReg:     reg,
		RootCtx:   netCtx.Background(),
		l:         l.AddCalldepth(1),
		lFields:   lf,
		stats:     s,
	}
}

func (c *context) L() log.Logger {
	return c.l
}

func (c *context) Stats() stats.Stats {
	return c.stats
}

func (c *context) Config() *config.Config {
	return c.AppConfig
}

func (c *context) BG() *bg.Reg {
	return c.BGReg
}

func (c *context) RootContext() netCtx.Context {
	return c.RootCtx
}

func (c *context) Trace(tag, msg string, fields ...log.Field) {
	c.l.Trace(tag, msg, c.lFields...)
	c.incLogLevelCount(log.LevelTrace, tag)
}

func (c *context) Warning(tag, msg string, fields ...log.Field) {
	c.l.Warning(tag, msg, c.lFields...)
	c.incLogLevelCount(log.LevelWarning, tag)
}

func (c *context) Error(tag, msg string, fields ...log.Field) {
	c.l.Error(tag, msg, c.lFields...)
	c.incLogLevelCount(log.LevelError, tag)
}

func (c *context) incLogLevelCount(lvl log.Level, tag string) {
	tags := map[string]string{
		"level":   lvl.String(),
		"tag":     tag,
		"service": c.Service,
		"node":    c.AppConfig.Node,
		"version": c.AppConfig.Version,
	}

	c.stats.Histogram("log.level", 1, tags)
}

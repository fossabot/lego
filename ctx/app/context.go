// Package app defines an application context, which carries information about
// the application environment.
//
// It can be information such as database credentials, service discovery,
// handlers addresses, ...
package app

import (
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
}

// context holds the application context
type context struct {
	Service   string
	AppConfig *config.Config
	BGReg     *bg.Reg

	l       log.Logger
	lFields []log.Field
	stats   stats.Stats
}

// NewCtx creates a new app context
func NewCtx(service string, c *config.Config, l log.Logger, s stats.Stats) Ctx {
	// Build background registry
	reg := bg.NewReg(l)

	lf := []log.Field{
		log.String("node", c.Node),
		log.String("version", c.Version),
		log.String("log_type", "A"),
	}

	return &context{
		Service:   service,
		AppConfig: c,
		BGReg:     reg,
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

func (c *context) Trace(tag, msg string, fields ...log.Field) {
	c.l.Trace(tag, msg, c.lFields...)
	c.incLogLevelCount(log.LevelTrace)
}

func (c *context) Warning(tag, msg string, fields ...log.Field) {
	c.l.Warning(tag, msg, c.lFields...)
	c.incLogLevelCount(log.LevelWarning)
}

func (c *context) Error(tag, msg string, fields ...log.Field) {
	c.l.Error(tag, msg, c.lFields...)
	c.incLogLevelCount(log.LevelError)
}

func (c *context) incLogLevelCount(lvl log.Level) {
	tags := map[string]string{
		"level":   lvl.String(),
		"service": c.Service,
		"node":    c.AppConfig.Node,
		"version": c.AppConfig.Version,
	}

	c.stats.Histogram("log.level", 1, tags)
}

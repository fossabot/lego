// Package app defines an application context, which carries information about
// the application environment.
//
// It can be information such as database credentials, service discovery,
// handlers addresses, ...
package app

import (
	"fmt"
	"strings"

	"github.com/stairlin/lego/bg"
	"github.com/stairlin/lego/config"
	"github.com/stairlin/lego/ctx"
	"github.com/stairlin/lego/log"
	"github.com/stairlin/lego/stats"
)

// Ctx is the app context interface
type Ctx interface {
	ctx.Ctx

	Name() string
	L() log.Logger
	Config() *config.Config
	BG() *bg.Reg
}

// context holds the application context
type context struct {
	AppName   string
	AppConfig *config.Config
	BGReg     *bg.Reg

	l     log.Logger
	stats stats.Stats
}

// NewCtx creates a new app context
func NewCtx(name string, c *config.Config, l log.Logger, s stats.Stats) Ctx {
	// Build background registry
	reg := bg.NewReg(l)

	return &context{
		AppName:   name,
		AppConfig: c,
		BGReg:     reg,
		l:         l,
		stats:     s,
	}
}

func (c *context) Name() string {
	return c.AppName
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

func (c *context) Trace(tag string, args ...interface{}) {
	c.l.Trace(spaceOut(c.logPrefix(), tag, spaceOut(args...)))
}

func (c *context) Tracef(tag string, format string, args ...interface{}) {
	c.l.Trace(spaceOut(c.logPrefix(), tag, fmt.Sprintf(format, args...)))
}

func (c *context) Warning(args ...interface{}) {
	c.l.Warning(spaceOut(c.logPrefix(), spaceOut(args...)))
}

func (c *context) Warningf(format string, args ...interface{}) {
	c.l.Warning(spaceOut(c.logPrefix(), fmt.Sprintf(format, args...)))
}

func (c *context) Error(args ...interface{}) {
	c.l.Error(spaceOut(c.logPrefix(), spaceOut(args...)))
}

func (c *context) Errorf(format string, args ...interface{}) {
	c.l.Error(spaceOut(c.logPrefix(), fmt.Sprintf(format, args...)))
}

func (c *context) logPrefix() string {
	return fmt.Sprintf("A %s", c.AppName)
}

// spaceOut joins the given args and separate them with spaces
func spaceOut(args ...interface{}) string {
	l := make([]string, len(args))
	for i, a := range args {
		l[i] = fmt.Sprint(a)
	}
	return strings.Join(l, " ")
}

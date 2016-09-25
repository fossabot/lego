// Package journey defines a context type, which carries information about
// a specific inbound request. It is created when it hits the first service
// and it is propagated accross all services.
//
// It has been named journey instead of request, because a journey can result
// of multiple sub-requests. And also because it sounds nice, isn't it?
package journey

import (
	"fmt"
	"strings"
	"sync"

	"github.com/satori/go.uuid"
	"github.com/stairlin/lego/bg"
	"github.com/stairlin/lego/config"
	"github.com/stairlin/lego/ctx"
	"github.com/stairlin/lego/ctx/app"
	"github.com/stairlin/lego/log"
	"github.com/stairlin/lego/stats"
)

// Ctx is the journey context interface
type Ctx interface {
	ctx.Ctx

	UUID() string
	ShortID() string
	AppConfig() *config.Config
	BG(f func()) error
}

// context holds the context of a request (journey) during its whole lifecycle
type context struct {
	mu sync.Mutex

	ID   string // (hopefuly) globally unique identifier
	Step uint
	app  app.Ctx
}

// New creates a new context and returns it
func New(ctx app.Ctx) Ctx {
	id := uuid.NewV4().String()

	return &context{
		ID:  id,
		app: ctx,
	}
}

// AppConfig returns the application configuration on which this context currently runs
func (c *context) AppConfig() *config.Config {
	return c.app.Config()
}

func (c *context) Stats() stats.Stats {
	return c.app.Stats()
}

// BG executes the given function in background
func (c *context) BG(f func()) error {
	return c.app.BG().Dispatch(bg.NewTask(f))
}

// UUID returns the universally unique identifier assigned to this context
func (c *context) UUID() string {
	return c.ID
}

// ShortID returns a partial representation of a request ID for the sake of readability
// However its uniqueness is not guarantee
func (c *context) ShortID() string {
	return strings.Split(c.ID, "-")[0]
}

func (c *context) Trace(tag string, args ...interface{}) {
	c.incTag(tag)
	c.log().Trace(buildLogLine(c.logPrefix(), tag, spaceOut(args...)))
}

func (c *context) Tracef(tag string, format string, args ...interface{}) {
	c.incTag(tag)
	c.log().Trace(buildLogLine(c.logPrefix(), tag, fmt.Sprintf(format, args...)))
}

func (c *context) Warning(args ...interface{}) {
	c.log().Warning(buildLogLine(c.logPrefix(), spaceOut(args...)))
}

func (c *context) Warningf(format string, args ...interface{}) {
	c.log().Warning(buildLogLine(c.logPrefix(), fmt.Sprintf(format, args...)))
}

func (c *context) Error(args ...interface{}) {
	c.log().Error(buildLogLine(c.logPrefix(), spaceOut(args...)))
}

func (c *context) Errorf(format string, args ...interface{}) {
	c.log().Error(buildLogLine(c.logPrefix(), fmt.Sprintf(format, args...)))
}

func (c *context) log() log.Logger {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Step++
	return c.l()
}

func (c *context) logPrefix() string {
	return fmt.Sprintf("J %s %s %04d", c.app.Name(), c.ShortID(), c.Step)
}

func (c *context) incTag(tag string) {
	tags := map[string]string{
		"tag": tag,
	}

	c.stats().Inc("log", tags)
}

func (c *context) l() log.Logger {
	return c.app.L()
}

func (c *context) stats() stats.Stats {
	return c.app.Stats()
}

// spaceOut joins the given args and separate them with spaces
func spaceOut(args ...interface{}) string {
	l := make([]string, len(args))
	for i, a := range args {
		l[i] = fmt.Sprint(a)
	}
	return strings.Join(l, " ")
}

func buildLogLine(l ...string) string {
	return strings.Join(l, " ")
}

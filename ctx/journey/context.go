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

	ID     string // (hopefuly) globally unique identifier
	Step   uint
	app    app.Ctx
	logger log.Logger
}

// New creates a new context and returns it
func New(ctx app.Ctx) Ctx {
	id := uuid.NewV4().String()

	// Log to correlate this journey with the current app environment
	ctx.Trace("ctx.journey.new", "Start journey",
		log.String("id", id),
	)

	return &context{
		ID:     id,
		app:    ctx,
		logger: ctx.L(),
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

func (c *context) Trace(tag, msg string, fields ...log.Field) {
	c.incTag(tag)
	c.log().Trace(tag, msg, c.logFields(fields)...)
	c.incLogLevelCount(log.LevelTrace, tag)
}

func (c *context) Warning(tag, msg string, fields ...log.Field) {
	c.incTag(tag)
	c.log().Warning(tag, msg, c.logFields(fields)...)
	c.incLogLevelCount(log.LevelWarning, tag)
}

func (c *context) Error(tag, msg string, fields ...log.Field) {
	c.incTag(tag)
	c.log().Error(tag, msg, c.logFields(fields)...)
	c.incLogLevelCount(log.LevelError, tag)
}

func (c *context) logFields(fields []log.Field) []log.Field {
	f := []log.Field{
		log.String("log_type", "J"),
		log.String("id", c.ShortID()),
		log.Uint("step", c.Step),
	}

	return append(f, fields...)
}

func (c *context) log() log.Logger {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Step++
	return c.logger
}

func (c *context) incTag(tag string) {
	tags := map[string]string{
		"tag": tag,
	}

	c.stats().Histogram("log", 1, tags)
}

func (c *context) incLogLevelCount(lvl log.Level, tag string) {
	tags := map[string]string{
		"level":   lvl.String(),
		"tag":     tag,
		"service": c.AppConfig().Service,
		"node":    c.AppConfig().Node,
		"version": c.AppConfig().Version,
	}

	c.app.Stats().Histogram("log.level", 1, tags)
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

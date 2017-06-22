package http

import (
	"net/http"
	"runtime/debug"
	"strconv"
	"time"

	"github.com/stairlin/lego/log"
)

// MiddlewareFunc is the function signature of a middelware
type MiddlewareFunc func(c *Context)

// Middleware is a function called on the HTTP stack before an action
type Middleware func(MiddlewareFunc) MiddlewareFunc

func buildMiddlewareChain(l []Middleware, action MiddlewareFunc) MiddlewareFunc {
	if len(l) == 0 {
		return action
	}

	c := action
	for i := len(l) - 1; i >= 0; i-- {
		c = l[i](c)
	}
	return c
}

// mwDebug adds useful debugging information to the response header
func mwDebug(next MiddlewareFunc) MiddlewareFunc {
	return func(c *Context) {
		c.Res.Header().Add("Request-Id", c.Ctx.UUID())
		next(c)
	}
}

// mwDraining blocks the request when the handler is draining
func mwDraining(next MiddlewareFunc) MiddlewareFunc {
	return func(c *Context) {
		if c.isDraining() {
			c.Ctx.Trace("http.mw.draining", "Service is draining")
			c.Res.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		next(c)
	}
}

// mwLogging logs information about HTTP requests/responses
func mwLogging(next MiddlewareFunc) MiddlewareFunc {
	return func(c *Context) {
		c.Ctx.Trace("h.http.req.start", "Request start",
			log.String("method", c.Method),
			log.String("path", c.Path),
			log.String("user_agent", c.Req.Header.Get("User-Agent")),
		)

		next(c)

		c.Ctx.Trace("h.http.req.end", "Request end",
			log.Int("status", c.Res.Code()),
			log.Duration("duration", time.Since(c.StartTime)),
		)
	}
}

// mwStats sends the request/response stats
func mwStats(next MiddlewareFunc) MiddlewareFunc {
	return func(c *Context) {
		tags := map[string]string{
			"method": c.Method,
			"path":   c.Path,
		}
		c.Ctx.Stats().Inc("http.conc", tags)

		// Next middleware
		next(c)

		tags["status"] = strconv.Itoa(c.Res.Code())
		c.Ctx.Stats().Histogram("http.call", 1, tags)
		c.Ctx.Stats().Timing("http.time", time.Since(c.StartTime), tags)
		c.Ctx.Stats().Dec("http.conc", tags)
	}
}

// mwPanic catches panic and recover
func mwPanic(next MiddlewareFunc) MiddlewareFunc {
	return func(c *Context) {
		// Wrap call to the next middleware
		func() {
			defer func() {
				if recover := recover(); recover != nil {
					c.Res.WriteHeader(http.StatusInternalServerError)
					c.Ctx.Error("http.mw.panic", "Recovered from panic",
						log.Object("err", recover),
						log.String("stack", string(debug.Stack())),
					)
				}
			}()

			next(c)
		}()
	}
}

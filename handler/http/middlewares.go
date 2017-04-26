package http

import (
	"net/http"
	"runtime/debug"
	"strconv"
	"time"

	"github.com/stairlin/lego/ctx/journey"
	"github.com/stairlin/lego/log"
)

// MiddlewareFunc is the function signature of a middelware
type MiddlewareFunc func(c *Context) int

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

func mwStartJourney(next MiddlewareFunc) MiddlewareFunc {
	return func(c *Context) int {
		// Assign unique request ID
		c.Ctx = journey.New(c.App)
		status := next(c)
		c.Ctx.End()
		return status
	}
}

// mwDebug adds useful debugging information to the response header
func mwDebug(next MiddlewareFunc) MiddlewareFunc {
	return func(c *Context) int {
		status := next(c)
		c.Res.Header().Add("Request-Id", c.Ctx.UUID())
		c.Res.Header().Add("Request-Duration", time.Since(c.StartTime).String())
		return status
	}
}

// mwDraining blocks the request when the handler is draining
func mwDraining(next MiddlewareFunc) MiddlewareFunc {
	return func(c *Context) int {
		if c.isDraining() {
			c.Ctx.Trace("http.mw.draining", "Service is draining")
			return http.StatusServiceUnavailable
		}
		return next(c)
	}
}

// mwDraining blocks the request when the handler is draining
func mwLogging(next MiddlewareFunc) MiddlewareFunc {
	return func(c *Context) int {
		c.Ctx.Trace("h.http.req.start", "Request start",
			log.String("method", c.Method),
			log.String("path", c.Path),
			log.String("user_agent", c.Req.Header.Get("User-Agent")),
		)

		status := next(c)

		c.Ctx.Trace("h.http.req.end", "Request end",
			log.Int("status", status),
			log.Duration("duration", time.Since(c.StartTime)),
		)
		return status
	}
}

// mwStats sends the request/response stats
func mwStats(next MiddlewareFunc) MiddlewareFunc {
	return func(c *Context) int {
		tags := map[string]string{
			"method": c.Method,
			"path":   c.Path,
		}
		c.Ctx.Stats().Inc("http.conc", tags)

		// Next middleware
		status := next(c)

		tags["status"] = strconv.Itoa(status)
		c.Ctx.Stats().Histogram("http.call", 1, tags)
		c.Ctx.Stats().Timing("http.time", time.Since(c.StartTime), tags)
		c.Ctx.Stats().Dec("http.conc", tags)

		return status
	}
}

// mwInterrupt returns the request when the context deadline expires or is being cancelled
func mwInterrupt(next MiddlewareFunc) MiddlewareFunc {
	return func(c *Context) int {
		res := make(chan int, 1)
		go func() {
			res <- next(c)
		}()

		select {
		case status := <-res:
			return status
		case <-c.Ctx.Done():
			c.Ctx.Trace("http.mw.interrupt", "Request cancelled or timed out", log.Error(c.Ctx.Err()))
			return http.StatusGatewayTimeout
		}
	}
}

// mwPanic catches panic and recover
func mwPanic(next MiddlewareFunc) MiddlewareFunc {
	return func(c *Context) int {
		var status int

		// Wrap call to the next middleware
		func() {
			defer func() {
				if recover := recover(); recover != nil {
					status = http.StatusInternalServerError
					c.Ctx.Error("http.mw.panic", "Recovered from panic",
						log.Object("err", recover),
						log.String("stack", string(debug.Stack())),
					)
				}
			}()

			status = next(c)
		}()

		return status
	}
}

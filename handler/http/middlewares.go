package http

import (
	"fmt"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/stairlin/lego/log"
)

// Middleware is a function called on the HTTP stack before an action
type Middleware func(CallFunc) CallFunc

func buildMiddlewareChain(l []Middleware, a Action) CallFunc {
	if len(l) == 0 {
		return a.Call
	}

	c := a.Call
	for i := len(l) - 1; i >= 0; i-- {
		c = l[i](c)
	}

	return c
}

// mwDraining blocks the request when the handler is draining
func mwDraining(next CallFunc) CallFunc {
	return func(c *Context) Renderer {
		if c.isDraining() {
			c.Ctx.Trace("http.mw.draining", "Service is draining")
			return c.Head(http.StatusServiceUnavailable)
		}
		return next(c)
	}
}

// mwInterrupt returns the request when the context deadline expires or is being cancelled
func mwInterrupt(next CallFunc) CallFunc {
	return func(c *Context) Renderer {
		res := make(chan Renderer, 1)
		go func() {
			res <- next(c)
		}()

		select {
		case r := <-res:
			return r
		case <-c.Ctx.Done():
			c.Ctx.Trace("http.mw.interrupt", "Request cancelled or timed out", log.Error(c.Ctx.Err()))
			return c.Head(http.StatusGatewayTimeout)
		}
	}
}

// mwDraining blocks the request when the handler is draining
func mwLogging(next CallFunc) CallFunc {
	return func(c *Context) Renderer {
		c.Ctx.Trace("h.http.req.start", "Request start",
			log.String("method", c.Req.Method),
			log.String("path", c.Req.URL.String()),
			log.String("user_agent", c.Req.Header.Get("User-Agent")),
		)

		r := next(c)

		c.Ctx.Trace("h.http.req.end", "Request end",
			log.Int("status", r.Status()),
			log.Duration("duration", time.Since(c.StartAt)),
		)

		return r
	}
}

// mwStats sends the request/response stats
func mwStats(next CallFunc) CallFunc {
	return func(c *Context) Renderer {
		tags := map[string]string{
			"action": fmt.Sprintf("%T", c.action),
		}
		c.Ctx.Stats().Inc("http.conc", tags)

		// Next middleware
		r := next(c)

		tags["status"] = fmt.Sprintf("%v", r.Status())
		c.Ctx.Stats().Histogram("http.call", 1, tags)
		c.Ctx.Stats().Timing("http.time", time.Since(c.StartAt), tags)
		c.Ctx.Stats().Dec("http.conc", tags)

		return r
	}
}

// mwPanic catches panic and recover
func mwPanic(next CallFunc) CallFunc {
	return func(c *Context) Renderer {
		p := false
		var r Renderer

		// Wrap call to the next middleware
		func() {
			defer func() {
				if err := recover(); err != nil {
					p = true
					c.Ctx.Error("http.mw.panic", "Recovered from panic",
						log.Object("err", err),
						log.String("stack", string(debug.Stack())),
					)
				}
			}()

			r = next(c)
		}()

		if p {
			return c.Head(http.StatusInternalServerError)
		}
		return r
	}
}

// mwRender encodes the response
func mwRender(next CallFunc) CallFunc {
	return func(c *Context) Renderer {
		r := next(c)

		// Encode response
		err := r.Encode(c)
		if err != nil {
			c.Ctx.Error("http.mw.render", "Renderer error", log.Error(err))
			c.Res.WriteHeader(http.StatusInternalServerError)
			return r
		}

		return r
	}
}

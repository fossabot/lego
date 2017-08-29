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
type MiddlewareFunc func(ctx journey.Ctx, w ResponseWriter, r *Request)

// Middleware is a function called on the HTTP stack before an action
type Middleware func(MiddlewareFunc) MiddlewareFunc

func buildMiddlewareChain(l []Middleware,
	action func(journey.Ctx, ResponseWriter, *Request),
) MiddlewareFunc {
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
	return func(ctx journey.Ctx, w ResponseWriter, r *Request) {
		w.Header().Add("Request-Id", ctx.UUID())
		next(ctx, w, r)
	}
}

// mwLogging logs information about HTTP requests/responses
func mwLogging(next MiddlewareFunc) MiddlewareFunc {
	return func(ctx journey.Ctx, w ResponseWriter, r *Request) {
		ctx.Trace("h.http.req.start", "Request start",
			log.String("method", r.method),
			log.String("path", r.path),
			log.String("user_agent", r.HTTP.Header.Get("User-Agent")),
		)

		next(ctx, w, r)

		ctx.Trace("h.http.req.end", "Request end",
			log.Int("status", w.Code()),
			log.Duration("duration", time.Since(r.startTime)),
		)
	}
}

// mwStats sends the request/response stats
func mwStats(next MiddlewareFunc) MiddlewareFunc {
	return func(ctx journey.Ctx, w ResponseWriter, r *Request) {
		tags := map[string]string{
			"method": r.method,
			"path":   r.path,
		}
		ctx.Stats().Inc("http.conc", tags)

		// Next middleware
		next(ctx, w, r)

		tags["status"] = strconv.Itoa(w.Code())
		ctx.Stats().Histogram("http.call", 1, tags)
		ctx.Stats().Timing("http.time", time.Since(r.startTime), tags)
		ctx.Stats().Dec("http.conc", tags)
	}
}

// mwPanic catches panic and recover
func mwPanic(next MiddlewareFunc) MiddlewareFunc {
	return func(ctx journey.Ctx, w ResponseWriter, r *Request) {
		// Wrap call to the next middleware
		func() {
			defer func() {
				if ctx.AppConfig().Request.Panic {
					return
				}
				if recover := recover(); recover != nil {
					w.WriteHeader(http.StatusInternalServerError)
					ctx.Error("http.mw.panic", "Recovered from panic",
						log.Object("err", recover),
						log.String("stack", string(debug.Stack())),
					)
				}
			}()

			next(ctx, w, r)
		}()
	}
}

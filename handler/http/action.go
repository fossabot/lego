package http

import (
	"net/http"

	"github.com/stairlin/lego/log"
)

// Action is an endpoint that handles incoming HTTP requests for a specific route.
// An action is stateless, self contained and should not share its context with other actions.
type Action interface {
	Call(c *Context) Renderer
}

// ActionFunc is the function signature of an action
type ActionFunc func(c *Context) Renderer

// renderActionFunc returns a func that executes the action and encodes the response with a renderer
func renderActionFunc(f ActionFunc) MiddlewareFunc {
	return func(c *Context) {
		res := make(chan Renderer, 1)
		go func() {
			res <- f(c)
		}()

		select {
		case r := <-res:
			if err := r.Render(c.Res, c.Req); err != nil {
				c.Ctx.Error("http.render", "Renderer error", log.Error(err))
				c.Res.WriteHeader(http.StatusInternalServerError)
			}
		case <-c.Ctx.Done():
			c.Ctx.Trace("http.interrupt", "Request cancelled or timed out", log.Error(c.Ctx.Err()))
			c.Res.WriteHeader(http.StatusGatewayTimeout)
		}
	}
}

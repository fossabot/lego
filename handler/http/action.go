package http

import (
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/stairlin/lego/ctx/app"
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
	return func(c *Context) int {
		renderer := f(c)
		if err := renderer.Encode(c); err != nil {
			c.Ctx.Error("http.render", "Renderer error", log.Error(err))
			return http.StatusInternalServerError
		}
		return renderer.Status()
	}
}

type bareHandler struct {
	method     string
	path       string
	a          ActionFunc
	app        app.Ctx
	isDraining func() bool
	add        func()
	done       func()
	call       MiddlewareFunc
}

func (h *bareHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	// Add to waitgroup
	h.add()
	defer h.done()

	// Build context
	c := &Context{
		App:       h.app,
		Ctx:       nil,
		StartTime: time.Now(),
		Params:    mux.Vars(r),
		Method:    h.method,
		Path:      h.path,
		Res:       rw,
		Req:       r,

		isDraining: h.isDraining,
		action:     h.a,
	}

	// Start call chain
	rw.WriteHeader(h.call(c))
}

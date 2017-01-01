package http

import (
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/stairlin/lego/ctx/app"
	"github.com/stairlin/lego/ctx/journey"
	"github.com/stairlin/lego/log"
)

// Action is an endpoint that handles incoming HTTP requests for a specific route.
// An action is stateless, self contained and should not share its context with other actions.
type Action interface {
	Call(c *Context) Renderer
}

// CallFunc is the contract required to be callable on the call chain
type CallFunc func(c *Context) Renderer

// Context holds the request context that is injected into an action
type Context struct {
	Ctx     journey.Ctx
	Res     http.ResponseWriter
	Req     *http.Request
	Parser  Parser
	Params  map[string]string
	StartAt time.Time

	isDraining func() bool
	action     Action
}

// JSON encodes the given data to JSON
func (c *Context) JSON(code int, data interface{}) Renderer {
	return &RenderJSON{Code: code, V: data}
}

// Head returns a body-less response
func (c *Context) Head(code int) Renderer {
	return &RenderHead{Code: code}
}

// Redirect returns an HTTP redirection response
func (c *Context) Redirect(url string) Renderer {
	return &RenderRedirect{URL: url}
}

type actionHandler struct {
	path       string
	method     string
	a          Action
	app        app.Ctx
	isDraining func() bool
	add        func()
	done       func()
	callChain  CallFunc
}

func (h *actionHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	// Add to waitgroup
	h.add()
	defer h.done()

	// Assign unique request ID
	journey := journey.New(h.app)
	defer journey.End()

	// Build context
	c := &Context{
		Ctx:        journey,
		Res:        rw,
		Req:        r,
		Params:     mux.Vars(r),
		StartAt:    time.Now(),
		isDraining: h.isDraining,
		action:     h.a,
	}

	// Pick decoder
	if hasBody(r.Method) {
		c.Parser = PickParser(journey, r)
	}

	// Add request ID to response header (useful for debugging)
	rw.Header().Add("X-Request-Id", journey.UUID())

	// Start call chain
	renderer := h.callChain(c)

	// Encode response
	err := renderer.Encode(c)
	if err != nil {
		journey.Error("action.encode.error", "Renderer error", log.Error(err))
		c.Res.WriteHeader(http.StatusInternalServerError)
		return
	}
}

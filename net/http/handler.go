package http

import (
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/mux"

	"github.com/stairlin/lego/ctx/app"
	"github.com/stairlin/lego/ctx/journey"
	"github.com/stairlin/lego/log"
)

const (
	// down mode is the default state. The handler is not ready to accept
	// new connections
	down uint32 = 0
	// up mode is when a handler accepts connections
	up uint32 = 1
	// drain mode is when a handler stops accepting new connection, but wait
	// for all existing in-flight requests to complete
	drain uint32 = 2
)

// Handler is a lego handler for the HTTP protocol
type Handler struct {
	wg   sync.WaitGroup
	mode uint32

	routes      []Route
	middlewares []Middleware
	static      struct {
		Path string
		Dir  string
	}
}

// NewHandler creates a new metal handler
func NewHandler() *Handler {
	h := &Handler{}

	// Register required middlewares
	h.Append(mwDebug)
	h.Append(mwDraining)
	h.Append(mwStats)
	h.Append(mwLogging)
	h.Append(mwPanic)

	return h
}

// Handle registers a new action on the given path and method
func (h *Handler) Handle(path, method string, a Action) {
	h.HandleFunc(path, method, a.Call)
}

// HandleFunc registers a new function as an action on the given path and method
func (h *Handler) HandleFunc(path, method string, f ActionFunc) {
	r := Route{
		Path:   path,
		Method: method,
		Action: f,
	}
	h.routes = append(h.routes, r)
}

// Append appends the given middleware to the call chain
func (h *Handler) Append(m Middleware) {
	h.middlewares = append(h.middlewares, m)
}

// Static registers a new route with path prefix to serve
// static files from the provided root directory.
func (h *Handler) Static(path, dir string) {
	h.static.Path = path
	h.static.Dir = dir
}

// Serve starts serving HTTP requests (blocking call)
func (h *Handler) Serve(addr string, ctx app.Ctx) error {
	// Map actions
	r := mux.NewRouter()
	for _, route := range h.routes {
		chain := buildMiddlewareChain(h.middlewares, renderActionFunc(route.Action))

		h := bareHandler{
			path:       route.Path,
			method:     route.Method,
			a:          route.Action,
			app:        ctx,
			isDraining: h.isDraining,
			add:        h.add,
			done:       h.done,
			call:       chain,
		}
		r.Handle(route.Path, &h).Methods(route.Method, OPTIONS)
	}

	// Map static directory (if any)
	if h.static.Path != "" && h.static.Dir != "" {
		ctx.Trace("h.http.static", "Serving static files...",
			log.String("path", h.static.Path),
			log.String("dir", h.static.Dir),
		)

		sh := staticHandler{
			App: ctx,
			FS:  &fileHandler{root: http.Dir(h.static.Dir)},
		}
		r.PathPrefix(h.static.Path).Handler(http.StripPrefix(h.static.Path, &sh))
	}

	ctx.Trace("h.http.listen", "Listening...", log.String("addr", addr))
	atomic.StoreUint32(&h.mode, up)
	return http.ListenAndServe(addr, r)
}

// Drain puts the handler into drain mode. All new requests will be
// blocked with a 503 and it will block this call until all in-flight requests
// have been completed
func (h *Handler) Drain() {
	atomic.StoreUint32(&h.mode, drain)
	h.wg.Wait() // Wait for all in-flight requests to complete
	atomic.StoreUint32(&h.mode, down)
}

// isDraining checks whether the handler is draining
func (h *Handler) isDraining() bool {
	return atomic.LoadUint32(&h.mode) == drain
}

// add signals a new inbound request
func (h *Handler) add() {
	h.wg.Add(1)
}

// done signals the end of a request
func (h *Handler) done() {
	h.wg.Done()
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

func (h *bareHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Add to waitgroup
	h.add()
	defer h.done()

	// Build context
	res := Response{http: w}
	c := Context{
		App:       h.app,
		StartTime: time.Now(),
		Params:    mux.Vars(r),
		Method:    h.method,
		Path:      h.path,
		Res:       &res,
		Req:       r,

		isDraining: h.isDraining,
		action:     h.a,
	}

	// Start or pick up journey
	if c.App.Config().Request.AllowContext && HasContext(c.Req) {
		// Pick up journey where downstream left off
		j, err := UnmarshalContext(c.App, c.Req)
		if err != nil {
			c.App.Warning("http.journey.parse.err", "Cannot parse journey", log.Error(err))
			c.Res.WriteHeader(http.StatusBadRequest)
			return
		}
		c.Ctx = j
	} else {
		// Assign unique request ID
		c.Ctx = journey.New(c.App)
	}

	// Start call chain
	h.call(&c)
	c.Ctx.End()
}

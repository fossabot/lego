package http

import (
	"net/http"
	"sync"

	"github.com/stairlin/lego/ctx/app"

	"github.com/gorilla/mux"
)

var bodyRequestMethods = []string{POST, PUT, DELETE}

func hasBody(m string) bool {
	for _, method := range bodyRequestMethods {
		if m == method {
			return true
		}
	}

	return false
}

// Handler is a lego handler for the HTTP protocol
type Handler struct {
	mu    sync.Mutex
	wg    sync.WaitGroup
	drain bool

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
	h.Append(mwPanic)
	h.Append(mwDraining)
	h.Append(mwStats)
	h.Append(mwLogging)

	return h
}

// Handle registers a new action on the given path and method
func (h *Handler) Handle(path, method string, a Action) {
	r := Route{
		Path:   path,
		Method: method,
		Action: a,
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
	// Print out middlewares and routes
	ctx.Infof("h.http.middlewares", "%d", len(h.middlewares))
	ctx.Infof("h.http.routes", "\n%s", table(h.routes))

	// Map actions
	r := mux.NewRouter()
	for _, route := range h.routes {
		chain := buildMiddlewareChain(h.middlewares, route.Action)

		h := &actionHandler{
			path:       route.Path,
			method:     route.Method,
			a:          route.Action,
			app:        ctx,
			isDraining: h.isDraining,
			add:        h.add,
			done:       h.done,
			callChain:  chain,
		}

		r.Handle(route.Path, h).Methods(route.Method, OPTIONS)
	}

	// Map static directory (if any)
	if h.static.Path != "" && h.static.Dir != "" {
		ctx.Infof("h.http.static", "Mapping %s with %s", h.static.Path, h.static.Dir)
		sh := &staticHandler{
			App: ctx,
			FS:  http.FileServer(http.Dir(h.static.Dir)),
		}
		r.PathPrefix(h.static.Path).Handler(http.StripPrefix(h.static.Path, sh))
	}

	ctx.Infof("h.http.listen", addr)
	return http.ListenAndServe(addr, r)
}

// Drain puts the handler into drain mode. All new requests will be
// blocked with a 503 and it will block this call until all in-flight requests
// have been completed
func (h *Handler) Drain() {
	h.mu.Lock()
	h.drain = true // Block all new requests
	h.mu.Unlock()

	h.wg.Wait() // Wait for all in-flight requests to complete
}

// isDraining checks whether the handler is draining
func (h *Handler) isDraining() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.drain
}

// add signals a new inbound request
func (h *Handler) add() {
	h.wg.Add(1)
}

// done signals the end of a request
func (h *Handler) done() {
	h.wg.Done()
}

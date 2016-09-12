package http

import (
	"net/http"
	"sync"
	"time"

	"github.com/stairlin/lego/ctx/app"
	"github.com/stairlin/lego/ctx/journey"

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

// Serve starts serving HTTP requests (blocking call)
func (h *Handler) Serve(addr string, ctx app.Ctx) error {
	// Print out middlewares and routes
	ctx.Infof("h.http.middlewares", "%d", len(h.middlewares))
	ctx.Infof("h.http.routes", "\n%s", table(h.routes))

	// Map actions
	r := mux.NewRouter()
	for _, route := range h.routes {
		chain := buildMiddlewareChain(h.middlewares, route.Action)

		h := &handler{
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

type handler struct {
	path       string
	method     string
	a          Action
	app        app.Ctx
	isDraining func() bool
	add        func()
	done       func()
	callChain  CallFunc
}

func (h *handler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	// Add to waitgroup
	h.add()
	defer h.done()

	// Assign unique request ID
	journey := journey.New(h.app)

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
		journey.Error("Renderer error", err)
		c.Res.WriteHeader(http.StatusInternalServerError)
		return
	}
}

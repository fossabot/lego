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
// TODO: Rename server
type Handler struct {
	wg   sync.WaitGroup
	mode uint32

	server http.Server

	endpoints   []Endpoint
	middlewares []Middleware

	certFile string
	keyFile  string
}

// NewHandler creates a new metal handler
func NewHandler() *Handler {
	h := &Handler{}

	// Register required middlewares
	h.Append(mwDebug)
	h.Append(mwStats)
	h.Append(mwLogging)
	h.Append(mwPanic)

	return h
}

// HandleFunc registers a new function as an action on the given path and method
func (h *Handler) HandleFunc(path, method string,
	f func(ctx journey.Ctx, w ResponseWriter, r *Request),
) {
	h.HandleEndpoint(&actionEndpoint{
		path:   path,
		method: method,
		call:   buildMiddlewareChain(h.middlewares, renderActionFunc(f)),
	})
}

// HandleStatic registers a new route on the given path with path prefix
// to serve static files from the provided root directory
func (h *Handler) HandleStatic(path, dir string) {
	h.HandleEndpoint(&fileEndpoint{
		path: path,
		fs:   &fileHandler{root: http.Dir(dir)},
	})
}

// HandleEndpoint registers an endpoint.
// This is particularily useful for custom endpoint types
func (h *Handler) HandleEndpoint(e Endpoint) {
	h.endpoints = append(h.endpoints, e)
}

// Append appends the given middleware to the call chain
func (h *Handler) Append(m Middleware) {
	h.middlewares = append(h.middlewares, m)
}

// ActivateTLS activates TLS on this handler. That means only incoming HTTPS
// connections are allowed.
//
// If the certificate is signed by a certificate authority, the certFile should
// be the concatenation of the server's certificate, any intermediates,
// and the CA's certificate.
func (h *Handler) ActivateTLS(certFile, keyFile string) {
	h.certFile = certFile
	h.keyFile = keyFile
}

// SetOptions changes the handler options
func (h *Handler) SetOptions(opts ...Option) {
	for _, opt := range opts {
		opt(h)
	}
}

// Serve starts serving HTTP requests (blocking call)
func (h *Handler) Serve(addr string, ctx app.Ctx) error {
	r := mux.NewRouter()
	for _, e := range h.endpoints {
		e.Attach(r, h.buildHandleFunc(ctx, e))
	}

	h.server.Addr = addr
	h.server.Handler = r

	tlsEnabled := h.certFile != "" && h.keyFile != ""
	ctx.Trace("h.http.listen", "Listening...", log.String("addr", addr),
		log.Bool("tls", tlsEnabled),
	)
	atomic.StoreUint32(&h.mode, up)
	if tlsEnabled {
		return h.server.ListenAndServeTLS(h.certFile, h.keyFile)
	}
	return h.server.ListenAndServe()
	// TODO: Handle listen errors and catch shutdown error
}

// Drain puts the handler into drain mode. All new requests will be
// blocked with a 503 and it will block this call until all in-flight requests
// have been completed
func (h *Handler) Drain() {
	atomic.StoreUint32(&h.mode, drain)
	h.wg.Wait() // Wait for all in-flight requests to complete
	atomic.StoreUint32(&h.mode, down)
	// TODO: server shutdown
}

// isDraining checks whether the handler is draining
func (h *Handler) isDraining() bool {
	return atomic.LoadUint32(&h.mode) == drain
}

func (h *Handler) buildHandleFunc(app app.Ctx, e Endpoint) func(
	w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// Add to waitgroup
		h.wg.Add(1)
		defer h.wg.Done()

		// Build context
		res := &responseWriter{http: w}
		req := &Request{
			startTime: time.Now(),
			method:    e.Method(),
			path:      e.Path(),

			HTTP:   r,
			Params: mux.Vars(r),
		}

		// Start or pick up journey
		var ctx journey.Ctx
		if app.Config().Request.AllowContext && HasContext(req.HTTP) {
			// Pick up journey where downstream left off
			j, err := UnmarshalContext(app, req.HTTP)
			if err != nil {
				app.Warning("http.journey.parse.err", "Cannot parse journey",
					log.Error(err),
				)
				w.WriteHeader(StatusBadRequest)
				return
			}
			ctx = j
		} else {
			// Assign unique request ID
			ctx = journey.New(app)
		}

		if h.isDraining() {
			ctx.Trace("http.draining", "Handler is draining")
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}

		// Handle request
		// TODO: Call middlewares from here and then call endpoint
		e.Handle(ctx, res, req)

		// Call it at the end (no defer)
		ctx.End()
	}
}

package http

import (
	"context"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/mux"

	"github.com/stairlin/lego/ctx/app"
	"github.com/stairlin/lego/ctx/journey"
	"github.com/stairlin/lego/log"
	"github.com/stairlin/lego/net"
)

// A Server defines parameters for running a lego compatible HTTP server
// The zero value for Server is a valid configuration.
type Server struct {
	wg    sync.WaitGroup
	state uint32

	http http.Server

	endpoints   []Endpoint
	middlewares []Middleware

	certFile string
	keyFile  string
}

// NewServer creates a new server and attaches the default middlewares
func NewServer() *Server {
	s := &Server{}
	s.Append(mwDebug)
	s.Append(mwStats)
	s.Append(mwLogging)
	s.Append(mwPanic)
	s.Append(mwInterrupt)
	return s
}

// HandleFunc registers a new function as an action on the given path and method
func (s *Server) HandleFunc(
	path,
	method string,
	f func(ctx journey.Ctx, w ResponseWriter, r *Request),
) {
	s.HandleEndpoint(&stdEndpoint{
		path:       path,
		method:     method,
		handleFunc: f,
	})
}

// HandleStatic registers a new route on the given path with path prefix
// to serve static files from the provided root directory
func (s *Server) HandleStatic(
	path,
	root string,
	hook ...func(ctx journey.Ctx, w ResponseWriter, r *Request, serveFile func()),
) {
	e := &fileEndpoint{
		path:        path,
		fileHandler: &fileHandler{root: http.Dir(root)},
	}
	if len(hook) > 0 {
		e.hook = hook[0]
	}
	s.HandleEndpoint(e)
}

// HandleEndpoint registers an endpoint.
// This is particularily useful for custom endpoint types
func (s *Server) HandleEndpoint(e Endpoint) {
	s.endpoints = append(s.endpoints, e)
}

// Append appends the given middleware to the call chain
func (s *Server) Append(m Middleware) {
	s.middlewares = append(s.middlewares, m)
}

// ActivateTLS activates TLS on this handler. That means only incoming HTTPS
// connections are allowed.
//
// If the certificate is signed by a certificate authority, the certFile should
// be the concatenation of the server's certificate, any intermediates,
// and the CA's certificate.
func (s *Server) ActivateTLS(certFile, keyFile string) {
	s.certFile = certFile
	s.keyFile = keyFile
}

// SetOptions changes the handler options
func (s *Server) SetOptions(opts ...Option) {
	for _, opt := range opts {
		opt(s)
	}
}

// Serve starts serving HTTP requests (blocking call)
func (s *Server) Serve(addr string, ctx app.Ctx) error {
	r := mux.NewRouter()
	for _, e := range s.endpoints {
		e.Attach(r, s.buildHandleFunc(ctx, e))
	}

	s.http.Addr = addr
	s.http.Handler = r

	tlsEnabled := s.certFile != "" && s.keyFile != ""
	ctx.Trace("s.http.listen", "Listening...", log.String("addr", addr),
		log.Bool("tls", tlsEnabled),
	)

	atomic.StoreUint32(&s.state, net.StateUp)
	var err error
	if tlsEnabled {
		err = s.http.ListenAndServeTLS(s.certFile, s.keyFile)
	}
	err = s.http.ListenAndServe()
	atomic.StoreUint32(&s.state, net.StateDown)

	if err == http.ErrServerClosed {
		// Suppress error caused by a server Shutdown or Close
		return nil
	}
	return err
}

// Drain puts the handler into drain mode. All new requests will be
// blocked with a 503 and it will block this call until all in-flight requests
// have been completed
func (s *Server) Drain() {
	atomic.StoreUint32(&s.state, net.StateDrain)
	s.wg.Wait()                           // Wait for all in-flight requests to complete
	s.http.Shutdown(context.Background()) // Then close all idle connections
}

// isState checks the current server state
func (s *Server) isState(state uint32) bool {
	return atomic.LoadUint32(&s.state) == uint32(state)
}

func (s *Server) buildHandleFunc(app app.Ctx, e Endpoint) func(
	w http.ResponseWriter, r *http.Request) {

	serve := buildMiddlewareChain(s.middlewares, e)

	return func(w http.ResponseWriter, r *http.Request) {
		// Add to waitgroup for a graceful shutdown
		s.wg.Add(1)
		defer s.wg.Done()

		// Wrap net/http parameters
		res := &responseWriter{http: w}
		req := &Request{
			startTime: time.Now(),
			method:    e.Method(),
			path:      e.Path(),

			HTTP:   r,
			Params: mux.Vars(r),
		}

		// Start or resume journey
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

		if s.isState(net.StateDrain) {
			ctx.Trace("http.draining", "Handler is draining")
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}

		// Handle request
		serve(ctx, res, req)

		// Call it at the end (no defer)
		ctx.End()
	}
}

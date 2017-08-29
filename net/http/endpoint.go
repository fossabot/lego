package http

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/stairlin/lego/ctx/journey"
)

type Endpoint interface {
	Path() string
	Method() string
	Attach(*mux.Router, func(http.ResponseWriter, *http.Request))
	Handle(journey.Ctx, ResponseWriter, *Request)
}

type actionEndpoint struct {
	method string
	path   string
	call   MiddlewareFunc
}

func (h *actionEndpoint) Path() string {
	return h.path
}

func (h *actionEndpoint) Method() string {
	return h.method
}

func (h *actionEndpoint) Attach(r *mux.Router, f func(http.ResponseWriter,
	*http.Request)) {
	r.HandleFunc(h.path, f).Methods(h.method, OPTIONS)
}

func (h *actionEndpoint) Handle(ctx journey.Ctx, w ResponseWriter, r *Request) {
	h.call(ctx, w, r)
}

type fileEndpoint struct {
	path string
	fs   *fileHandler
}

func (h *fileEndpoint) Path() string {
	return h.path
}

func (h *fileEndpoint) Method() string {
	return GET
}

func (h *fileEndpoint) Attach(r *mux.Router, f func(http.ResponseWriter,
	*http.Request)) {
	// r.PathPrefix(h.path).Handler(http.StripPrefix(h.path, f))
	// FIXME: Fix path/prefix/dir stuff
}

func (h *fileEndpoint) Handle(ctx journey.Ctx, w ResponseWriter, r *Request) {
	h.fs.ServeHTTP(w, r.HTTP)
}

package http

import (
	"net/http"

	"github.com/stairlin/lego/ctx/app"
	"github.com/stairlin/lego/ctx/journey"
)

type staticHandler struct {
	App app.Ctx
	FS  *fileHandler
}

func (h *staticHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	j := journey.New(h.App)
	j.Tracef("h.http.static.inbound", "File: <%s>", r.URL)
	j.Tracef("h.http.static.cache", "If-Modified-Since: <%s>", r.Header.Get("If-Modified-Since"))

	// Wrap response writer to get the status code
	res := &responseWriter{ResponseWriter: rw, status: http.StatusOK}

	h.FS.ServeHTTP(res, r)

	j.Tracef("h.http.static.res", "Status: <%d>", res.status)
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(status int) {
	rw.status = status
	rw.ResponseWriter.WriteHeader(status)
}

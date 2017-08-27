package http

import (
	"net/http"

	"github.com/stairlin/lego/ctx/app"
	"github.com/stairlin/lego/ctx/journey"
	"github.com/stairlin/lego/log"
)

type staticHandler struct {
	App app.Ctx
	FS  *fileHandler
}

func (h *staticHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	j := journey.New(h.App)
	j.Trace("h.http.static.start", "Serve static file",
		log.String("path", r.URL.String()),
	)
	j.Trace("h.http.static.cache", "Caching headers",
		log.String("if_modified_since", r.Header.Get("If-Modified-Since")),
	)

	// Wrap response writer to get the status code
	res := &responseWriter{ResponseWriter: rw, status: http.StatusOK}

	h.FS.ServeHTTP(res, r)

	j.Trace("h.http.static.end", "Done", log.Int("status", res.status))
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(status int) {
	rw.status = status
	rw.ResponseWriter.WriteHeader(status)
}

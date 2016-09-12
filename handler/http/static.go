package http

import (
	"net/http"

	"github.com/stairlin/lego/ctx/app"
)

type staticHandler struct {
	App app.Ctx
	FS  http.Handler
}

func (h *staticHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	h.App.Infof("h.http.static.serve", "File: <%s> If-Modified-Since: <%s>",
		r.URL,
		r.Header.Get("If-Modified-Since"),
	)

	h.FS.ServeHTTP(rw, r)
}

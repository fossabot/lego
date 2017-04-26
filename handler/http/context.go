package http

import (
	"net/http"
	"time"

	"github.com/stairlin/lego/ctx/app"
	"github.com/stairlin/lego/ctx/journey"
)

// Context holds the request context that is injected into an action
type Context struct {
	App       app.Ctx
	Ctx       journey.Ctx
	StartTime time.Time
	Params    map[string]string
	Method    string
	Path      string
	Res       http.ResponseWriter
	Req       *http.Request

	isDraining func() bool
	action     ActionFunc
}

// Parse parses the request body and decodes it on the given struct
func (c *Context) Parse(v interface{}) error {
	return pickParser(c.Ctx, c.Req).Parse(v)
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

package http

import (
	"encoding/json"
	"net/http"
)

// Renderer is a response returned by an action
type Renderer interface {
	// Status returns the HTTP response status code
	Status() int
	// Encode marshals the response
	Encode(*Context) error
}

// RenderJSON is a renderer that marshal responses in JSON
type RenderJSON struct {
	Code int
	V    interface{}
}

func (r *RenderJSON) Status() int {
	return r.Code
}

func (r *RenderJSON) Encode(ctx *Context) error {
	// Header
	ctx.Res.Header().Add("Content-Type", "application/json; charset=utf-8")
	ctx.Res.WriteHeader(r.Code)

	// Body
	json, err := json.Marshal(r.V)
	if err != nil {
		return err
	}

	_, err = ctx.Res.Write(json)
	if err != nil {
		return err
	}

	return nil
}

// RenderHead is a renderer that returns a body-less response
type RenderHead struct {
	Code int
}

func (r *RenderHead) Status() int {
	return r.Code
}

func (r *RenderHead) Encode(ctx *Context) error {
	ctx.Res.WriteHeader(r.Code)

	return nil
}

// RenderRedirect is a renderer that returns a redirection
type RenderRedirect struct {
	URL string
}

func (r *RenderRedirect) Status() int {
	return http.StatusTemporaryRedirect
}

func (r *RenderRedirect) Encode(ctx *Context) error {
	http.Redirect(ctx.Res, ctx.Req, r.URL, r.Status())

	return nil
}

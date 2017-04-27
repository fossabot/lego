package http

import (
	"encoding/json"
	"io"
	"net/http"
	"time"
)

// Renderer is a response returned by an action
type Renderer interface {
	Render(http.ResponseWriter, *http.Request) error
}

// RenderJSON is a renderer that marshal responses in JSON
type RenderJSON struct {
	Code int
	V    interface{}
}

func (r *RenderJSON) Render(res http.ResponseWriter, req *http.Request) error {
	// Header
	res.Header().Add("Content-Type", "application/json; charset=utf-8")
	res.WriteHeader(r.Code)

	// Body
	if err := json.NewEncoder(res).Encode(r.V); err != nil {
		return err
	}
	return nil
}

// RenderHead is a renderer that returns a body-less response
type RenderHead struct {
	Code int
}

func (r *RenderHead) Render(res http.ResponseWriter, req *http.Request) error {
	res.WriteHeader(r.Code)
	return nil
}

// RenderRedirect is a renderer that returns a redirection
type RenderRedirect struct {
	URL string
}

func (r *RenderRedirect) Render(res http.ResponseWriter, req *http.Request) error {
	http.Redirect(res, req, r.URL, http.StatusTemporaryRedirect)
	return nil
}

type RenderData struct {
	ContentType string
	Reader      io.ReadCloser
}

func (r *RenderData) Render(res http.ResponseWriter, req *http.Request) error {
	defer r.Reader.Close()
	if r.ContentType != "" {
		res.Header()["Content-Type"] = []string{r.ContentType}
	}
	_, err := io.Copy(res, r.Reader)
	return err
}

type RenderConditional struct {
	ETag         string
	LastModified time.Time
	Renderer     Renderer
}

func (r *RenderConditional) Render(res http.ResponseWriter, req *http.Request) error {
	lastModified := r.LastModified.UTC().Format(http.TimeFormat)
	if r.ETag != "" {
		res.Header().Add("ETag", r.ETag)
	}
	if !r.LastModified.IsZero() {
		res.Header().Add("Last-Modified", lastModified)
	}

	if (r.ETag != "" && req.Header.Get("If-None-Match") == r.ETag) ||
		(lastModified != "" && req.Header.Get("If-Modified-Since") == lastModified) {
		res.WriteHeader(http.StatusNotModified)
		return nil
	}
	return r.Render(res, req) // Call next renderer
}

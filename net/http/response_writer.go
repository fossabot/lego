package http

import (
	"io"
	"net/http"
	"sync"
	"time"
)

// ResponseWriter wraps the standard net/http Response struct
// The main reason of wrapping it is to interact with the status code
// and avoid double the “http: multiple response.WriteHeader calls” warnings
type ResponseWriter interface {
	// Header returns the header map that will be sent by
	// WriteHeader. The Header map also is the mechanism with which
	// Handlers can set HTTP trailers.
	//
	// Changing the header map after a call to WriteHeader (or
	// Write) has no effect unless the modified headers are
	// trailers.
	//
	// There are two ways to set Trailers. The preferred way is to
	// predeclare in the headers which trailers you will later
	// send by setting the "Trailer" header to the names of the
	// trailer keys which will come later. In this case, those
	// keys of the Header map are treated as if they were
	// trailers. See the example. The second way, for trailer
	// keys not known to the Handler until after the first Write,
	// is to prefix the Header map keys with the TrailerPrefix
	// constant value. See TrailerPrefix.
	//
	// To suppress implicit response headers (such as "Date"), set
	// their value to nil.
	Header() http.Header

	// Write writes the data to the connection as part of an HTTP reply.
	//
	// If WriteHeader has not yet been called, Write calls
	// WriteHeader(http.StatusOK) before writing the data. If the Header
	// does not contain a Content-Type line, Write adds a Content-Type set
	// to the result of passing the initial 512 bytes of written data to
	// DetectContentType.
	//
	// Depending on the HTTP protocol version and the client, calling
	// Write or WriteHeader may prevent future reads on the
	// Request.Body. For HTTP/1.x requests, handlers should read any
	// needed request body data before writing the response. Once the
	// headers have been flushed (due to either an explicit Flusher.Flush
	// call or writing enough data to trigger a flush), the request body
	// may be unavailable. For HTTP/2 requests, the Go HTTP server permits
	// handlers to continue to read the request body while concurrently
	// writing the response. However, such behavior may not be supported
	// by all HTTP/2 clients. Handlers should read before writing if
	// possible to maximize compatibility.
	Write([]byte) (int, error)

	// WriteHeader sends an HTTP response header with status code.
	// If WriteHeader is not called explicitly, the first call to Write
	// will trigger an implicit WriteHeader(http.StatusOK).
	// Thus explicit calls to WriteHeader are mainly used to
	// send error codes.
	WriteHeader(int)

	// Code returns the written status code.
	// If it has not been set yet, it will return 0
	Code() int

	// HasCode returns whether the status code has been set
	HasCode() bool

	// JSON encodes the given data to JSON
	JSON(code int, data interface{}) error

	// Gob encodes the given data to Gob
	Gob(code int, data interface{}) error

	// Head returns a body-less response
	Head(code int) error

	// Redirect returns an HTTP redirection response
	Redirect(req *http.Request, url string) error

	// Data encodes an arbitrary type of data
	Data(code int, contentType string, data io.ReadCloser) error

	// Conditional checks whether the request conditions are fresh.
	// If the request is fresh, it returns a 304, otherwise it calls the next renderer
	Conditional(req *http.Request, etag string, lastModified time.Time, next func() error) error
}

// responseWriter is the implementation of ResponseWriter
type responseWriter struct {
	mu          sync.RWMutex
	http        http.ResponseWriter
	code        int
	codeWritten bool
}

func (r *responseWriter) Header() http.Header {
	return r.http.Header()
}

func (r *responseWriter) Write(b []byte) (int, error) {
	return r.http.Write(b)
}

func (r *responseWriter) WriteHeader(c int) {
	r.mu.Lock()
	if !r.codeWritten {
		r.code = c
		r.codeWritten = true
		r.http.WriteHeader(c)
	}
	r.mu.Unlock()
}

func (r *responseWriter) Code() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.code
}

func (r *responseWriter) HasCode() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.codeWritten
}

func (r *responseWriter) JSON(code int, data interface{}) error {
	f := &RenderJSON{Code: code, V: data}
	return f.Render(r)
}

func (r *responseWriter) Gob(code int, data interface{}) error {
	f := &RenderGob{Code: code, V: data}
	return f.Render(r)
}

func (r *responseWriter) Head(code int) error {
	f := &RenderHead{Code: code}
	return f.Render(r)
}

func (r *responseWriter) Redirect(req *http.Request, url string) error {
	f := &RenderRedirect{Req: req, URL: url}
	return f.Render(r)
}

func (r *responseWriter) Data(code int, contentType string, data io.ReadCloser,
) error {
	f := &RenderData{Code: code, ContentType: contentType, Reader: data}
	return f.Render(r)
}

func (r *responseWriter) Conditional(req *http.Request, etag string,
	lastModified time.Time, next func() error) error {
	f := &RenderConditional{
		Req:          req,
		ETag:         etag,
		LastModified: lastModified,
		Next:         &rendererWrapper{F: next},
	}
	return f.Render(r)
}

type rendererWrapper struct {
	F func() error
}

func (r *rendererWrapper) Render(ResponseWriter) error {
	return r.F()
}

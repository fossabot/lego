package http

import (
	"net/http"
	"sync"
)

// Response wraps the standard net/http Response struct
// The main reason of wrapping it is to interact with the status code
// and avoid double the “http: multiple response.WriteHeader calls” warnings
type Response struct {
	mu          sync.RWMutex
	http        http.ResponseWriter
	code        int
	codeWritten bool
}

// Header returns the header map that will be sent by
// WriteHeader. The Header map also is the mechanism with which
// Handlers can set HTTP trailers.
//
// Changing the header map after a call to WriteHeader (or
// Write) has no effect unless the modified headers are
// trailers.
func (r *Response) Header() http.Header {
	return r.http.Header()
}

// Write writes the data to the connection as part of an HTTP reply.
//
// If WriteHeader has not yet been called, Write calls
// WriteHeader(http.StatusOK) before writing the data. If the Header
// does not contain a Content-Type line, Write adds a Content-Type set
// to the result of passing the initial 512 bytes of written data to
// DetectContentType.
func (r *Response) Write(b []byte) (int, error) {
	return r.http.Write(b)
}

// WriteHeader sends an HTTP response header with status code.
// If WriteHeader is not called explicitly, the first call to Write
// will trigger an implicit WriteHeader(http.StatusOK).
// Thus explicit calls to WriteHeader are mainly used to
// send error codes.
func (r *Response) WriteHeader(c int) {
	r.mu.Lock()
	if !r.codeWritten {
		r.code = c
		r.codeWritten = true
		r.http.WriteHeader(c)
	}
	r.mu.Unlock()
}

// Code returns the written status code.
// If it has not been set yet, it will return 0
func (r *Response) Code() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.code
}

// HasCode returns whether the status code has been set
func (r *Response) HasCode() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.codeWritten
}

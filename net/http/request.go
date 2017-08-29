package http

import (
	"net/http"
	"time"

	"github.com/stairlin/lego/ctx/journey"
)

// Request wraps the standard net/http Request struct
type Request struct {
	startTime time.Time
	method    string
	path      string

	HTTP   *http.Request
	Params map[string]string
}

// Parse parses the request body and decodes it on the given struct
func (r *Request) Parse(ctx journey.Ctx, v interface{}) error {
	return pickParser(ctx, r).Parse(v)
}

package http

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"mime"
	"net/http"

	"github.com/stairlin/lego/ctx/journey"
	"github.com/stairlin/lego/log"
)

const (
	mimeJSON = "application/json"
)

// Parser is a request data Parser
type Parser interface {
	Type() string
	Parse(v interface{}) error
}

// PickParser selects a Parser for the request content-type
func PickParser(ctx journey.Ctx, req *http.Request) Parser {
	ct := req.Header.Get("Content-Type")
	m, _, err := mime.ParseMediaType(req.Header.Get("Content-Type"))
	if err != nil {
		// This can probably be demoted to a warning at some point
		ctx.Error("http.content_type.err", "Cannot parse Content-Type",
			log.String("content_type", ct),
			log.Error(err),
		)
	}

	switch m {
	case mimeJSON:
		ctx.Trace("action.parser", "Pick JSON parser", log.String("type", mimeJSON))
		return &ParseJSON{req}
	}

	ctx.Trace("action.parser", "Pick null parser", log.String("type", "null"))
	return &ParseNull{m}
}

// ParseJSON Parses JSON
type ParseJSON struct {
	req *http.Request
}

// Type returns the mime type
func (d *ParseJSON) Type() string {
	return mimeJSON
}

// Parse unmarshal the request payload into the given structure
func (d *ParseJSON) Parse(v interface{}) error {
	data, err := ioutil.ReadAll(d.req.Body)
	if err != nil {
		return err
	}
	defer d.req.Body.Close()

	return json.Unmarshal(data, v)
}

// ParseNull is a null-object that is used when no other Parsers have been found
type ParseNull struct {
	mime string
}

// Type returns the mime type
func (d *ParseNull) Type() string {
	return ""
}

// Parse returns an error because this Parser has nothing to Parse
func (d *ParseNull) Parse(v interface{}) error {
	return fmt.Errorf("ParseNull: %s", d.mime)
}

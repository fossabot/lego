package http

import (
	"encoding/base64"
	"net/http"

	"github.com/pkg/errors"
	"github.com/stairlin/lego/ctx/app"
	"github.com/stairlin/lego/ctx/journey"
)

const contextHeader = "Ctx-Journey"

// HasContext checks whether the request contains a context
func HasContext(req *http.Request) bool {
	return req.Header.Get(contextHeader) != ""
}

// UnmarshalContext unmarshal a context from the request
func UnmarshalContext(app app.Ctx, req *http.Request) (journey.Ctx, error) {
	header := req.Header.Get(contextHeader)
	data, err := base64.StdEncoding.DecodeString(string(header))
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal context")
	}
	ctx, err := journey.UnmarshalGob(app, []byte(data))
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal Context")
	}
	ctx = ctx.BranchOff(journey.Child)
	return ctx, nil
}

// MarshalContext marshals a context to the request.
func MarshalContext(ctx journey.Ctx, req *http.Request) error {
	data, err := journey.MarshalGob(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to marshal Context")
	}
	text := base64.StdEncoding.EncodeToString(data)
	req.Header.Add(contextHeader, text)
	return nil
}

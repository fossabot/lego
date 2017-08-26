package journey

import (
	"bytes"
	"encoding/gob"

	"github.com/pkg/errors"

	"github.com/stairlin/lego/ctx/app"
)

// MarshalGob marshals a context in gob
func MarshalGob(c Ctx) ([]byte, error) {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(c); err != nil {
		return nil, errors.Wrap(err, "failed to marshal context")
	}
	return buf.Bytes(), nil
}

// UnmarshalGob unmarshals a gob encoded context
func UnmarshalGob(ctx app.Ctx, data []byte) (Ctx, error) {
	c := build(ctx)
	buf := bytes.NewBuffer(data)
	if err := gob.NewDecoder(buf).Decode(c); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal context")
	}
	return c, nil
}

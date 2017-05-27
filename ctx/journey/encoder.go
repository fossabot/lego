package journey

import (
	"encoding/base64"
	"strings"

	"github.com/pkg/errors"
)

const (
	textSeparator = "."
)

type textEncoder struct {
	enc *base64.Encoding
}

func newTextEncoder() *textEncoder {
	return &textEncoder{
		enc: base64.StdEncoding,
	}
}

func (e *textEncoder) Encode(l ...string) []byte {
	parts := make([]string, len(l))
	for i, part := range l {
		encoded := e.enc.EncodeToString([]byte(part))
		parts[i] = encoded
	}
	return []byte(strings.Join(parts, textSeparator))
}

func (e *textEncoder) Decode(b []byte) ([]string, error) {
	subs := strings.Split(string(b), textSeparator)
	parts := make([]string, len(subs))
	for i, sub := range subs {
		part, err := e.enc.DecodeString(sub)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot decode part %d", i)
		}
		parts[i] = string(part)
	}
	return parts, nil
}

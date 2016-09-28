package json

import (
	"encoding/json"

	"github.com/stairlin/lego/log"
)

const Name = "json"

func New(c map[string]string) (log.Formatter, error) {
	return &Formatter{}, nil
}

type Formatter struct{}

func (f *Formatter) Format(ctx *log.Ctx, tag, msg string, fields ...log.Field) (string, error) {
	out := &out{
		Level:     ctx.Level,
		Timestamp: ctx.Timestamp,
		Service:   ctx.Service,
		File:      ctx.File,
		Tag:       tag,
		Msg:       msg,
		Fields:    formatFields(fields),
	}

	r, err := json.Marshal(out)
	if err != nil {
		return "", err
	}

	return string(r), nil
}

func formatFields(fields []log.Field) []kv {
	l := make([]kv, len(fields))
	for i, f := range fields {
		k, v := f.KV()
		l[i] = kv{K: k, V: v}
	}
	return l
}

type out struct {
	Level     string `json:"level"`
	Timestamp string `json:"timestamp"`
	Service   string `json:"service"`
	File      string `json:"file"`
	Type      string `json:"type"`
	Tag       string `json:"tag,omitempty"`
	Msg       string `json:"msg"`
	Fields    []kv   `json:"fields"`
}

type kv struct {
	K string `json:"k"`
	V string `json:"v"`
}

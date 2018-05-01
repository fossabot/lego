// Package file prints log lines to a file.
package file

import (
	"io"
	"os"

	"github.com/pkg/errors"
	"github.com/stairlin/lego/config"
	"github.com/stairlin/lego/log"
)

const (
	Name = "file"

	defaultMode = 0660
	flag        = os.O_CREATE | os.O_WRONLY | os.O_APPEND
)

// Config defines the filer printer config
type Config struct {
	Path string `toml:"path"`
	Flag int    `toml:"flag"`
	Mode uint32 `toml:"mode"`
}

func New(tree config.Tree) (log.Printer, error) {
	c := Config{}
	if err := tree.Unmarshal(&c); err != nil {
		return nil, err
	}
	if c.Path == "" {
		return nil, errors.New("missing \"path\" on file log printer config")
	}
	if c.Mode == 0 {
		c.Mode = defaultMode
	}

	f, err := os.OpenFile(c.Path, flag, os.FileMode(c.Mode))
	if err != nil {
		return nil, errors.Wrap(err, "failed to open log file")
	}

	return &Logger{
		W: f,
	}, nil
}

type Logger struct {
	W io.WriteCloser
}

func (l *Logger) Print(ctx *log.Ctx, s string) error {
	_, err := l.W.Write([]byte(s))
	return err
}

func (l *Logger) Close() error {
	return l.W.Close()
}

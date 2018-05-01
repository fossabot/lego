// Package file prints log lines to a file.
package file

import (
	"bufio"
	"os"
	"sync"

	"github.com/pkg/errors"
	"github.com/stairlin/lego/config"
	"github.com/stairlin/lego/log"
)

const (
	Name = "file"

	defaultMode = 0660
	flag        = os.O_CREATE | os.O_WRONLY | os.O_APPEND
	newLine     = '\n'
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
	buf := bufio.NewWriter(f)

	return &Logger{
		buf:  buf,
		file: f,
	}, nil
}

type Logger struct {
	mu sync.Mutex

	buf  *bufio.Writer
	file *os.File
}

func (l *Logger) Print(ctx *log.Ctx, s string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	_, err := l.buf.WriteString(s)
	if err != nil {
		return err
	}
	return l.buf.WriteByte(newLine)
}

func (l *Logger) Close() error {
	l.buf.Flush()
	return l.file.Close()
}

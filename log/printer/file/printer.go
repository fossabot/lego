// Package file prints log lines to a file.
package file

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

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

	l := &Logger{
		conf:   c,
		sighup: make(chan os.Signal, 1),
	}
	go l.listen()
	return l, l.open()
}

type Logger struct {
	mu sync.Mutex

	conf   Config
	buf    *bufio.Writer
	file   *os.File
	sighup chan os.Signal
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
	l.mu.Lock()
	defer l.mu.Unlock()

	signal.Stop(l.sighup)
	close(l.sighup)

	return l.close()
}

func (l *Logger) open() (err error) {
	l.file, err = os.OpenFile(l.conf.Path, flag, os.FileMode(l.conf.Mode))
	if err != nil {
		return errors.Wrap(err, "failed to open log file")
	}
	l.buf = bufio.NewWriter(l.file)
	return nil
}

func (l *Logger) close() error {
	l.buf.Flush()
	return l.file.Close()
}

// listen listens to SIGHUP signals to reopen the log file.
// Logrotated can be configured to send a SIGHUP signal to a process after
// rotating it's logs.
func (l *Logger) listen() {
	signal.Notify(l.sighup, syscall.SIGHUP)
	for range l.sighup {
		l.mu.Lock()
		defer l.mu.Unlock()

		fmt.Fprintf(os.Stderr, "%s: Reopening %q\n", time.Now(), l.conf.Path)
		if err := l.close(); err != nil {
			fmt.Fprintf(os.Stderr, "%s: Error closing log file: %s\n", time.Now(), err)
		}
		if err := l.open(); err != nil {
			fmt.Fprintf(os.Stderr, "%s: Error opening log file: %s\n", time.Now(), err)
		}
	}
}

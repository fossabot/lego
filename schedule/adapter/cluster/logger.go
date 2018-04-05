package cluster

import (
	"bufio"
	"bytes"
	"io"

	"github.com/stairlin/lego/ctx"
)

// logger is an ugly workaround the lack of logger interface on Hashicorp Raft
// This logger plugs on the io.Writer interface and parses the data to log them
// properly via the Logger interface
type logger struct {
	l    ctx.Logger
	r    *io.PipeReader
	w    *io.PipeWriter
	stop chan struct{}

	prefix string
}

func newLogger(l ctx.Logger, prefix string) *logger {
	r, w := io.Pipe()
	return &logger{
		l:      l,
		r:      r,
		w:      w,
		prefix: prefix,
	}
}

func (l *logger) Start() {
	l.stop = make(chan struct{})

	buf := bufio.NewReader(l.r)
	go func() {
		for {
			select {
			case <-l.stop:
				return
			default:
				p, err := buf.ReadBytes('\n')
				if err != nil {
					continue
				}
				r := bytes.Split(p, []byte("]"))
				if len(r) == 1 {
					continue
				}
				r[1] = bytes.TrimSpace(r[1])
				switch {
				case bytes.Contains(r[0], []byte("DEBUG")):
					l.l.Trace("s.schedule."+l.prefix+".debug", string(r[1]))
				case bytes.Contains(r[0], []byte("INFO")):
					l.l.Trace("s.schedule."+l.prefix+".info", string(r[1]))
				case bytes.Contains(r[0], []byte("WARN")):
					l.l.Warning("s.schedule."+l.prefix+".warn", string(r[1]))
				case bytes.Contains(r[0], []byte("ERR")):
					l.l.Warning("s.schedule."+l.prefix+".err", string(r[1]))
				default:
					l.l.Warning("s.schedule."+l.prefix+".unknown_level", string(r[1]))
				}
			}
		}
	}()
}

func (l *logger) Stop() {
	l.w.Close()
	l.r.Close()
	l.stop <- struct{}{}
	close(l.stop)
}

func (l *logger) Write(p []byte) (n int, err error) {
	return l.w.Write(p)
}

package handler

import (
	"errors"
	"fmt"
	"sync"

	"github.com/stairlin/lego/ctx/app"
)

// ErrEmptyReg is the error returned when there are no handlers registered
var ErrEmptyReg = errors.New("there must be at least one registered handler")

// H is the interface to implement to be a valid handler
type H interface {
	Serve(addr string, ctx app.Ctx) error
	Drain()
}

// Reg (registry) holds a list of H
type Reg struct {
	mu sync.Mutex

	ctx   app.Ctx
	l     map[string]H
	drain bool
}

// NewReg builds a new registry
func NewReg(ctx app.Ctx) *Reg {
	return &Reg{
		ctx: ctx,
		l:   map[string]H{},
	}
}

// Add adds the given handler to the list of handlers
func (r *Reg) Add(addr string, h H) {
	r.mu.Lock()
	defer r.mu.Unlock()

	err := r.register(addr, h)
	if err != nil {
		// If we attempt to register on the same address, we can assume it is a
		// config error, therefore we should fail loudly and as fast as possible,
		// hence the panic.
		panic(err)
	}
}

// Serve allows handlers to serve requests
func (r *Reg) Serve() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.l) == 0 {
		return ErrEmptyReg
	}

	r.ctx.Trace("handler.serve.init", "Starting handlers...")

	wg := sync.WaitGroup{}
	wg.Add(len(r.l))
	for addr, h := range r.l {
		go func(addr string, h H) {
			// Deregister itself upon completion
			defer func() {
				r.ctx.Tracef("lego.serve.h.stop", "Handler <addr:%s> <h:%T> has stopped running", addr, h)
				r.mu.Lock()
				r.deregister(addr)
				r.mu.Unlock()
			}()

			r.ctx.Tracef("lego.serve.h", "<addr:%s> <h:%T>", addr, h)
			wg.Done()
			err := h.Serve(addr, r.ctx)
			if err != nil {
				r.ctx.Errorf("Error with handler <h:%T> (%s)", h, err)
			}
		}(addr, h)
	}

	wg.Wait() // Wait to boot all handlers
	r.ctx.Tracef("handler.serve.ready", "All handlers are running")

	return nil
}

// Drain notify all handlers to enter in draining mode. It means they are no
// longer accepting new requests, but they can finish all in-flight requests
func (r *Reg) Drain() {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if we are already draining
	if r.drain {
		return
	}

	// Flag registry as draining
	r.drain = true

	// Build WG
	l := len(r.l)
	wg := sync.WaitGroup{}
	wg.Add(l)

	// Drain handlers
	r.ctx.Tracef("handler.drain.init", "Start draining... <%d>", l)
	for _, h := range r.l {
		r.ctx.Tracef("handler.drain.h", "%T", h)
		go func(h H) {
			h.Drain()
			wg.Done()
		}(h)
	}

	wg.Wait()

	r.drain = false
	r.ctx.Tracef("handler.drain.done", "All handlers have been drained")
}

func (r *Reg) register(addr string, h H) error {
	if _, ok := r.l[addr]; ok {
		return fmt.Errorf(
			"handler listening on <%s> has already been registered (%T)",
			addr,
			r.l[addr],
		)
	}

	r.l[addr] = h
	return nil
}

func (r *Reg) deregister(addr string) {
	delete(r.l, addr)
}

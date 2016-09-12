package http

import (
	"testing"

	lt "github.com/stairlin/lego/testing"
)

func TestBuildMiddlewares(t *testing.T) {
	tt := lt.New(t)
	factory := &mwFactory{t: tt}

	l := []Middleware{
		factory.newMiddleware(0),
		factory.newMiddleware(1),
		factory.newMiddleware(2),
	}
	a := &dummyAction{}

	c := buildMiddlewareChain(l, a)

	c(&Context{})

	expected := 3
	if factory.C != expected {
		tt.Errorf("expect to be have %d middlewares called, but got %d", expected, factory.C)
	}
}

type mwFactory struct {
	N int
	C int
	t *lt.T
}

func (f *mwFactory) newMiddleware(expected int) Middleware {
	n := f.N
	f.N++

	return func(next CallFunc) CallFunc {
		return func(c *Context) Renderer {
			f.C++
			if n != expected {
				f.t.Errorf("expect to be called in position %d, but got %d", expected, n)
			}
			return next(c)
		}
	}
}

type dummyRenderer struct {
}

func (r *dummyRenderer) Status() int           { return StatusOK }
func (r *dummyRenderer) Encode(*Context) error { return nil }

type dummyAction struct {
}

func (a *dummyAction) Call(c *Context) Renderer {
	return &dummyRenderer{}
}

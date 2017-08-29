package http

import (
	"testing"

	"github.com/stairlin/lego/ctx/journey"
	lt "github.com/stairlin/lego/testing"
)

func TestBuildMiddlewares(t *testing.T) {
	tt := lt.New(t)
	factory := &mwFactory{t: tt}
	appCtx := tt.NewAppCtx("test-middlewares")

	l := []Middleware{
		factory.newMiddleware(0),
		factory.newMiddleware(1),
		factory.newMiddleware(2),
	}
	a := func(ctx journey.Ctx, w ResponseWriter, r *Request) {}

	c := buildMiddlewareChain(l, renderActionFunc(a))

	c(journey.New(appCtx), &responseWriter{}, &Request{})

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

	return func(next MiddlewareFunc) MiddlewareFunc {
		return func(ctx journey.Ctx, w ResponseWriter, r *Request) {
			f.C++
			if n != expected {
				f.t.Errorf("expect to be called in position %d, but got %d", expected, n)
			}
			next(ctx, w, r)
		}
	}
}

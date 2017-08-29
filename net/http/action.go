package http

import (
	"net/http"
	"runtime/debug"

	"github.com/stairlin/lego/ctx/journey"
	"github.com/stairlin/lego/log"
)

// Action is an endpoint that handles incoming HTTP requests for a specific route.
// An action is stateless, self contained and should not share its context with other actions.
// type Action interface {
// 	Call(ctx journey.Ctx, w ResponseWriter, r *Request)
// }
//
// // ActionFunc is the function signature of an action
// type ActionFunc func(ctx journey.Ctx, w ResponseWriter, r *Request)

// renderActionFunc returns a func that executes the action and encodes the response with a renderer
func renderActionFunc(f func(ctx journey.Ctx, w ResponseWriter,
	r *Request)) MiddlewareFunc {
	return func(ctx journey.Ctx, w ResponseWriter, r *Request) {
		res := make(chan struct{}, 1)
		rec := make(chan interface{}, 1)

		go func() {
			defer func() {
				if ctx.AppConfig().Request.Panic {
					return
				}
				if recover := recover(); recover != nil {
					ctx.Error("http.panic", "Recovered from panic",
						log.Object("err", recover),
						log.String("stack", string(debug.Stack())),
					)
					rec <- recover
				}
			}()

			f(ctx, w, r)
			res <- struct{}{}
		}()

		select {
		case <-res:
			// OK
		case <-rec:
			// action panicked
			w.WriteHeader(http.StatusInternalServerError)
		case <-ctx.Done():
			ctx.Trace("http.interrupt", "Request cancelled or timed out", log.Error(ctx.Err()))
			w.WriteHeader(http.StatusGatewayTimeout)
		}
	}
}

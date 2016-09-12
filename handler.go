package lego

import "github.com/stairlin/lego/handler"

// RegisterHandler adds the given handler to the list of handlers
func (a *App) RegisterHandler(addr string, h handler.H) {
	a.handlers.Add(addr, h)
}

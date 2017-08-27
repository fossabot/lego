package lego

import "github.com/stairlin/lego/net"

// RegisterHandler adds the given handler to the list of handlers
func (a *App) RegisterHandler(addr string, h net.Handler) {
	a.handlers.Add(addr, h)
}

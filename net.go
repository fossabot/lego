package lego

import "github.com/stairlin/lego/net"

// RegisterServer adds the given server to the list of managed servers
func (a *App) RegisterServer(addr string, s net.Server) {
	a.servers.Add(addr, s)
}

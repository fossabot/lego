package schedule

import (
	"github.com/stairlin/lego/disco"
)

// Option allows to configure unexported handler fields
type Option func(*Server)

// OptDisco specifies a service discovery service that keeps the list of peers
// up to date.
// Each update from the service watcher is proposed to the cluster by the leader
func OptDisco(svc disco.Service) Option {
	return func(s *Server) {
		s.service = svc
	}
}

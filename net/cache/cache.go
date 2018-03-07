// Package cache allows to run a cache server that can connect to its own peers.
//
// The server shards by key to select which peer is responsible for that key.
// It also comes with a cache filling mechanism.
package cache

import (
	"context"
	"net/http"
	"strings"
	"sync/atomic"

	"github.com/golang/groupcache"
	"github.com/stairlin/lego/cache"
	"github.com/stairlin/lego/ctx/app"
	"github.com/stairlin/lego/log"
	"github.com/stairlin/lego/net"
	"github.com/stairlin/lego/net/naming"
)

// Server implements net.Server for distributed caching
type Server struct {
	state   uint32
	watcher naming.Watcher
	nodes   map[string]struct{}
	pool    *groupcache.HTTPPool
	opts    groupcache.HTTPPoolOptions

	HTTP  http.Server
	Cache cache.Cache
}

// NewServer initialises a new Server
func NewServer(cache cache.Cache) *Server {
	return &Server{
		nodes: map[string]struct{}{},
		opts: groupcache.HTTPPoolOptions{
			BasePath: "/",
			Replicas: 50,
		},
		Cache: cache,
	}
}

// NewGroup implements cache.Cache
func (s *Server) NewGroup(
	name string, cacheBytes int64, loader cache.LoadFunc,
) cache.Group {
	return s.Cache.NewGroup(name, cacheBytes, loader)
}

// Serve implements net.Server
func (s *Server) Serve(addr string, ctx app.Ctx) error {
	// Mount group cache to mux
	s.pool = groupcache.NewHTTPPoolOpts("http://"+addr, &s.opts)
	r := http.NewServeMux()
	r.Handle(s.opts.BasePath, s.pool)

	// Attach mux
	s.HTTP.Addr = addr
	s.HTTP.Handler = r

	// Watch for updates
	if s.watcher != nil {
		go func() {
			for {
				updates, err := s.watcher.Next()
				switch err {
				case nil:
				case naming.ErrWatcherClosed:
					return
				default:
					ctx.Warning("s.cache.watch.err", "Watcher returned an error",
						log.Error(err),
					)
				}

				for _, u := range updates {
					switch u.Op {
					case naming.Add:
						s.nodes[u.Addr] = struct{}{}
					case naming.Delete:
						delete(s.nodes, u.Addr)
					}
				}

				peers := make([]string, 0, len(s.nodes))
				for n := range s.nodes {
					peers = append(peers, s.buildURL(n))
				}
				s.pool.Set(peers...)
			}
		}()
	}

	ctx.Trace("s.cache.listen", "Listening...", log.String("addr", addr))
	atomic.StoreUint32(&s.state, net.StateUp)
	err := s.HTTP.ListenAndServe()
	atomic.StoreUint32(&s.state, net.StateDown)

	if err == http.ErrServerClosed {
		// Suppress error caused by a server Shutdown or Close
		return nil
	}
	return err
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// When the inbound request is not in the base path, groupcache panics
	if !strings.HasPrefix(r.URL.Path, s.opts.BasePath) {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	s.pool.ServeHTTP(w, r)
}

// Drain implements net.Server
func (s *Server) Drain() {
	s.HTTP.Shutdown(context.Background()) // Close all idle connections
	if s.watcher != nil {
		s.watcher.Close()
	}
}

// SetOptions changes the handler options
func (s *Server) SetOptions(opts ...Option) {
	for _, opt := range opts {
		opt(s)
	}
}

func (s *Server) isState(state uint32) bool {
	return atomic.LoadUint32(&s.state) == state
}

func (s *Server) buildURL(node string) string {
	// TODO: Handle TLS
	return "http://" + node
}

// Option allows to configure unexported handler fields
type Option func(*Server)

// OptBasePath specifies the HTTP path that will serve cache requests.
// If blank, it defaults to "/".
func OptBasePath(p string) Option {
	return func(s *Server) {
		s.opts.BasePath = p
	}
}

// OptReplicas specifies the number of key replicas on the consistent hash.
// If blank, it defaults to 50.
func OptReplicas(r int) Option {
	return func(s *Server) {
		s.opts.Replicas = r
	}
}

// OptWatcher specifies a watcher to keep the list of peers up to date
func OptWatcher(w naming.Watcher) Option {
	return func(s *Server) {
		s.watcher = w
	}
}

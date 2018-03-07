// Package consul is a wrapper around the Hashicorp Consul service discovery functionnality
//
// Consul is a highly available and distributed service discovery and key-value store designed
// with support for the modern data center to make distributed systems and configuration easy.
package consul

import (
	"context"
	"sort"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"github.com/stairlin/lego/config"
	"github.com/stairlin/lego/ctx"
	"github.com/stairlin/lego/disco"
	"github.com/stairlin/lego/log"
)

// Name contains the adapter registered name
const Name = "consul"

// New returns a new file config store
func New(c *config.Config, params map[string]string) (disco.Agent, error) {
	// Configure client
	cc := api.DefaultConfig()
	cc.Address = config.ValueOf(params["address"])
	cc.Datacenter = config.ValueOf(params["dc"])
	cc.Token = config.ValueOf(params["token"])

	// Build Consul client
	consul, err := api.NewClient(cc)
	if err != nil {
		return nil, errors.Wrap(err, "cannot initialise Consul client")
	}

	return &Agent{
		consul:       consul,
		consulConfig: cc,
		appConfig:    c,
		serviceIDs:   map[string]struct{}{},
		advertAddr:   config.ValueOf(params["advertise_address"]),
	}, nil
}

type Agent struct {
	mu sync.Mutex

	consul       *api.Client
	consulConfig *api.Config
	appConfig    *config.Config
	advertAddr   string

	// serviceIDs caches the list of services registered
	serviceIDs map[string]struct{}
}

func (a *Agent) Register(ctx ctx.Ctx, r *disco.Registration) (string, error) {
	tags := append(a.appConfig.Disco.DefaultTags, r.Tags...)

	ctx.Trace("disco.register", "Register service",
		log.String("name", r.Name),
		log.Uint("port", uint(r.Port)),
		log.String("adapter", "consul"),
		log.String("tags", strings.Join(tags, ", ")),
	)

	reg := api.AgentServiceRegistration{
		ID:      uuid.New().String(),
		Name:    r.Name,
		Port:    int(r.Port),
		Address: r.Addr,
		Tags:    tags,
	}
	if reg.Address == "" {
		reg.Address = a.advertAddr
	}
	err := a.consul.Agent().ServiceRegister(&reg)
	if err != nil {
		return "", err
	}

	a.mu.Lock()
	a.serviceIDs[reg.ID] = struct{}{}
	a.mu.Unlock()
	return reg.ID, nil
}

func (a *Agent) Deregister(ctx ctx.Ctx, id string) error {
	ctx.Trace("disco.deregister", "Deregister service",
		log.String("id", id),
		log.String("adapter", "consul"),
	)

	err := a.consul.Agent().ServiceDeregister(id)
	if err != nil {
		return err
	}

	a.mu.Lock()
	delete(a.serviceIDs, id)
	a.mu.Unlock()
	return nil
}

func (a *Agent) Services(
	ctx ctx.Ctx, tags ...string,
) (map[string]disco.Service, error) {
	r, err := a.consul.Agent().Services()
	if err != nil {
		return nil, err
	}

	svcs := map[string]disco.Service{}
	for id, s := range r {
		if !isSubset(s.Tags, tags) {
			continue
		}

		v, ok := svcs[s.Service]
		if !ok {
			v = &service{
				name: s.Service,
				watch: func() disco.Watcher {
					return a.watch(ctx, s.Service, tags...)
				},
			}
			svcs[s.Service] = v
		}

		svc := v.(*service)
		svc.instances = append(svc.instances, &disco.Instance{
			ID:   id,
			Name: s.Service,
			Host: s.Address,
			Port: uint16(s.Port),
			Tags: s.Tags,
		})
	}
	return svcs, nil
}

func (a *Agent) Service(
	ctx ctx.Ctx, name string, tags ...string,
) (disco.Service, error) {
	q := a.buildQueryOptions()
	instances, _, err := a.service(ctx, name, q, tags...)
	if err != nil {
		return nil, err
	}
	return &service{
		name:      name,
		instances: instances,
		watch: func() disco.Watcher {
			return a.watch(ctx, name, tags...)
		},
	}, nil
}

func (a *Agent) Leave(ctx ctx.Ctx) {
	for id := range a.serviceIDs {
		err := a.Deregister(ctx, id)
		if err != nil {
			ctx.Warning("disco.leave.failure", "Could not de-register service",
				log.String("service_id", id),
			)
		}
	}

	a.mu.Lock()
	a.serviceIDs = map[string]struct{}{}
	a.mu.Unlock()
}

func (a *Agent) watch(
	ctx context.Context, name string, tags ...string,
) disco.Watcher {
	var waitIndex uint64
	next := func(ctx context.Context) ([]*disco.Instance, error) {
		q := a.buildQueryOptions()
		q.WaitIndex = waitIndex
		instances, meta, err := a.service(ctx, name, q, tags...)
		if err != nil {
			return nil, err
		}
		waitIndex = meta.LastIndex
		return instances, nil
	}
	return newWatcher(ctx, next)
}

func (a *Agent) service(
	ctx context.Context, name string, q *api.QueryOptions, tags ...string,
) ([]*disco.Instance, *api.QueryMeta, error) {
	var tag string
	if len(tags) > 0 {
		tag = tags[0]
	}

	r, meta, err := a.consul.Health().Service(name, tag, true, q)
	if err != nil {
		return nil, nil, err
	}

	var instances []*disco.Instance
	for _, chk := range r {
		if !isSubset(chk.Service.Tags, tags) {
			continue
		}

		instances = append(instances, &disco.Instance{
			ID:   chk.Service.ID,
			Name: chk.Service.Service,
			Host: chk.Service.Address,
			Port: uint16(chk.Service.Port),
			Tags: chk.Service.Tags,
		})
	}
	return instances, meta, nil
}

func (a *Agent) buildQueryOptions() *api.QueryOptions {
	return &api.QueryOptions{
		Datacenter: a.consulConfig.Datacenter,
		Token:      a.consulConfig.Token,
	}
}

// service implements disco.Service
type service struct {
	name      string
	instances []*disco.Instance
	watch     func() disco.Watcher
}

func (s *service) Name() string {
	return s.name
}

func (s *service) Watch() disco.Watcher {
	return s.watch()
}

func (s *service) Instances() []*disco.Instance {
	return s.instances
}

type watcher struct {
	next   func(context.Context) ([]*disco.Instance, error)
	ctx    context.Context
	cancel func()

	diff disco.Diff
}

func newWatcher(
	ctx context.Context, n func(context.Context) ([]*disco.Instance, error),
) *watcher {
	child, cancelFunc := context.WithCancel(ctx)
	return &watcher{
		next:   n,
		ctx:    child,
		cancel: cancelFunc,
	}
}

func (w *watcher) Next() ([]*disco.Event, error) {
	if w.ctx.Err() != nil {
		return nil, disco.ErrWatcherClosed
	}

	instances, err := w.next(w.ctx)
	if err != nil {
		return nil, err
	}
	return w.diff.Apply(instances), nil
}

func (w *watcher) Close() error {
	w.cancel()
	return nil
}

// isSubset returns whether b is a subset of a
func isSubset(a, b []string) bool {
	if len(a) < len(b) {
		return false
	}
	if len(b) == 0 {
		return true
	}

	sort.Strings(a)
	sort.Strings(b)
	var matches int
	for i := 0; i < len(a); i++ {
		if a[i] == b[matches] {
			matches++
		}
		if matches == len(b) {
			return true
		}
	}
	return false
}

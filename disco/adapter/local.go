package adapter

import (
	"errors"
	"sync"

	"github.com/google/uuid"
	"github.com/stairlin/lego/ctx"
	"github.com/stairlin/lego/disco"
	"github.com/stairlin/lego/log"
)

// localAgent is a local-only service discovery agent
// This agent is used when service discovery is disabled
type localAgent struct {
	mu sync.RWMutex

	Registry map[string]*disco.Instance
	// subs contains all event subscriptions
	Subs map[chan *disco.Event]struct{}
}

func newLocalAgent() disco.Agent {
	return &localAgent{
		Registry: map[string]*disco.Instance{},
		Subs:     map[chan *disco.Event]struct{}{},
	}
}

func (a *localAgent) Register(ctx ctx.Ctx, r *disco.Registration) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	id := uuid.New().String()
	if _, ok := a.Registry[id]; ok {
		return "", errors.New("service already registered")
	}
	instance := &disco.Instance{
		Local: true,
		ID:    id,
		Name:  r.Name,
		Host:  r.Addr,
		Port:  r.Port,
		Tags:  r.Tags,
	}
	a.Registry[id] = instance

	// Notifiy subscribers
	for sub := range a.Subs {
		sub <- &disco.Event{
			Op:       disco.Add,
			Instance: instance,
		}
	}

	return id, nil
}

func (a *localAgent) Deregister(ctx ctx.Ctx, id string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	return a.deregister(ctx, id)
}

func (a *localAgent) Services(
	ctx ctx.Ctx, tags ...string,
) (map[string]disco.Service, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return a.services(ctx, tags...)
}

func (a *localAgent) Service(
	ctx ctx.Ctx, name string, tags ...string,
) (disco.Service, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	services, err := a.services(ctx, tags...)
	if err != nil {
		return nil, err
	}
	s, ok := services[name]
	if !ok {
		return nil, errors.New("service does not exist")
	}
	return s, nil
}

func (a *localAgent) Leave(ctx ctx.Ctx) {
	a.mu.Lock()
	defer a.mu.Unlock()

	for id := range a.Registry {
		err := a.deregister(ctx, id)
		if err != nil {
			ctx.Warning("disco.leave.failure", "Could not de-register service",
				log.String("service_id", id),
			)
		}
	}
	a.Registry = map[string]*disco.Instance{}
	a.Subs = map[chan *disco.Event]struct{}{}
}

func (a *localAgent) services(
	ctx ctx.Ctx, tags ...string,
) (map[string]disco.Service, error) {
	services := map[string]disco.Service{}
	for _, instance := range a.Registry {
		services[instance.Name] = &service{
			name:      instance.Name,
			instances: []*disco.Instance{instance},
			watch: func() disco.Watcher {
				sub := make(chan *disco.Event, 1)
				unsub := func() {
					delete(a.Subs, sub)
				}
				a.Subs[sub] = struct{}{}
				return &watcher{
					sub:   sub,
					unsub: unsub,
				}
			},
		}
	}
	return services, nil
}

func (a *localAgent) deregister(ctx ctx.Ctx, id string) error {
	instance, ok := a.Registry[id]
	if !ok {
		return nil
	}
	delete(a.Registry, id)

	// Notifiy subscribers
	for sub := range a.Subs {
		sub <- &disco.Event{
			Op:       disco.Delete,
			Instance: instance,
		}
	}

	return nil
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
	sub   chan *disco.Event
	unsub func()
}

func (w *watcher) Next() ([]*disco.Event, error) {
	e, ok := <-w.sub
	if !ok {
		return nil, disco.ErrWatcherClosed
	}
	return []*disco.Event{e}, nil
}

func (w *watcher) Close() error {
	w.unsub()
	close(w.sub)
	return nil
}

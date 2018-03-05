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
	id := uuid.New().String()
	if _, ok := a.Registry[id]; ok {
		return "", errors.New("service already registered")
	}
	a.Registry[id] = &disco.Instance{
		ID:   id,
		Name: r.Name,
		Host: r.Addr,
		Port: r.Port,
		Tags: r.Tags,
	}
	return "", nil
}

func (a *localAgent) Deregister(ctx ctx.Ctx, id string) error {
	delete(a.Registry, id)
	return nil
}

func (a *localAgent) Services(
	ctx ctx.Ctx, tags ...string,
) (map[string]disco.Service, error) {
	services := map[string]disco.Service{}
	for _, instance := range a.Registry {
		services[instance.Name] = &service{
			name:      instance.Name,
			instances: []*disco.Instance{instance},
			sub: func() (sub chan *disco.Event, unsub func()) {
				sub = make(chan *disco.Event)
				unsub = func() {
					delete(a.Subs, sub)
				}
				a.Subs[sub] = struct{}{}
				return sub, unsub
			},
		}
	}
	return services, nil
}

func (a *localAgent) Service(
	ctx ctx.Ctx, name string, tags ...string,
) (disco.Service, error) {
	services, err := a.Services(ctx, tags...)
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
		err := a.Deregister(ctx, id)
		if err != nil {
			ctx.Warning("disco.leave.failure", "Could not de-register service",
				log.String("service_id", id),
			)
		}
	}
	a.Registry = map[string]*disco.Instance{}
	a.Subs = map[chan *disco.Event]struct{}{}
}

// service implements disco.Service
type service struct {
	name      string
	instances []*disco.Instance
	sub       func() (sub chan *disco.Event, unsub func())
}

func (s *service) Name() string {
	return s.name
}

func (s *service) Sub() (sub chan *disco.Event, unsub func()) {
	return s.sub()
}

func (s *service) Instances() []*disco.Instance {
	return s.instances
}

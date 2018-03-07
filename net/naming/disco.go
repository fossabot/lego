package naming

import (
	"github.com/stairlin/lego/ctx"
	"github.com/stairlin/lego/disco"
)

// Disco creates a Disco Resolver that uses service discovery to find
// available instances.
// It also creates watchers that listen to service discovery updates
func Disco(ctx ctx.Ctx, disco disco.Agent, tags ...string) Resolver {
	return &discoResolver{
		ctx:   ctx,
		disco: disco,
		tags:  tags,
	}
}

type discoResolver struct {
	ctx   ctx.Ctx
	disco disco.Agent
	tags  []string
}

func (r *discoResolver) Resolve(target string) (Watcher, error) {
	svc, err := r.disco.Service(r.ctx, target, r.tags...)
	if err != nil {
		return nil, err
	}
	return &discoWatcher{w: svc.Watch()}, nil
}

type discoWatcher struct {
	w         disco.Watcher
	instances map[string]*disco.Instance
}

func (w *discoWatcher) Next() ([]*Update, error) {
	if w.instances == nil {
		w.instances = map[string]*disco.Instance{}
	}
	events, err := w.w.Next()
	switch err {
	case nil:
	case disco.ErrWatcherClosed:
		return nil, ErrWatcherClosed
	default:
		return nil, err
	}

	var updates []*Update
	for _, evt := range events {
		switch evt.Op {
		case disco.Add:
			w.instances[evt.Instance.ID] = evt.Instance
			updates = append(updates, &Update{
				Op:   Add,
				Addr: evt.Instance.Addr(),
			})
		case disco.Update:
			inst, ok := w.instances[evt.Instance.ID]
			if !ok {
				// TODO: Is this a realistic scenario?
				continue
			}

			// In case the address has changed
			if inst.Addr() != evt.Instance.Addr() {
				updates = append(updates,
					&Update{
						Op:   Delete,
						Addr: inst.Addr(),
					},
					&Update{
						Op:   Add,
						Addr: evt.Instance.Addr(),
					},
				)
			}
		case disco.Delete:
			delete(w.instances, evt.Instance.ID)
			updates = append(updates, &Update{
				Op:   Delete,
				Addr: evt.Instance.Addr(),
			})
		}
	}
	return updates, nil
}

func (w *discoWatcher) Close() error {
	return w.w.Close()
}

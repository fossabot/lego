package naming

import (
	"net/url"

	"github.com/stairlin/lego/ctx/app"
	"github.com/stairlin/lego/disco"
)

// Disco creates a Disco Resolver that uses service discovery to find
// available instances.
// It also creates watchers that listen to service discovery updates
func Disco(ctx app.Ctx, tags ...string) Resolver {
	return &discoResolver{
		ctx:  ctx,
		tags: tags,
	}
}

func buildDisco(ctx app.Ctx, uri *url.URL) (Watcher, error) {
	var tags []string
	for _, t := range uri.Query()["tag"] {
		if t != "" {
			tags = append(tags, t)
		}
	}
	return Disco(ctx, tags...).Resolve(uri.Host)
}

type discoResolver struct {
	ctx  app.Ctx
	tags []string
}

func (r *discoResolver) Resolve(target string) (Watcher, error) {
	svc, err := r.ctx.Disco().Service(r.ctx, target, r.tags...)
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

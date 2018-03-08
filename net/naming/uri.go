package naming

import "github.com/stairlin/lego/ctx/app"

// URI returns a resolver that uses the target URI scheme to select a real resolver
func URI(ctx app.Ctx) Resolver {
	return &uriResolver{ctx: ctx}
}

type uriResolver struct {
	ctx app.Ctx
}

// Resolve creates a Watcher for target.
func (r *uriResolver) Resolve(target string) (Watcher, error) {
	return Resolve(r.ctx, target)
}

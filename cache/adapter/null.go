package adapter

import (
	"github.com/stairlin/lego/cache"
	"github.com/stairlin/lego/ctx/journey"
)

// nullCache is a cache which does not do anything.
type nullCache struct{}

func newNullCache() cache.Cache {
	return &nullCache{}
}

func (c *nullCache) NewGroup(
	name string, cacheBytes int64, loader cache.LoadFunc,
) cache.Group {
	return &group{load: loader}
}

type group struct {
	load cache.LoadFunc
}

func (g *group) Get(ctx journey.Ctx, key string) ([]byte, error) {
	return g.load(ctx, key)
}

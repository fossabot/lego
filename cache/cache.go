// Package cache is a caching and cache-filling library
package cache

import (
	"context"

	"github.com/golang/groupcache"
	"github.com/stairlin/lego/ctx"
	"github.com/stairlin/lego/log"
)

type Cache interface {
	// NewGroup creates a LRU caching namespace with a size limit and a load
	// function to be called when the value is mising
	NewGroup(name string, cacheBytes int64, loader LoadFunc) Group
}

// A Group is a cache namespace
type Group interface {
	Get(ctx context.Context, key string) ([]byte, error)
}

// A LoadFunc loads data for a key.
type LoadFunc func(context context.Context, key string) ([]byte, error)

// New initialises a new cache store
func New() Cache {
	return &gcache{
		groups: map[string]Group{},
	}
}

// gcache is a tiny wrapper for groupcache
type gcache struct {
	groups map[string]Group
}

func (c *gcache) NewGroup(name string, cacheBytes int64, loader LoadFunc) Group {
	if g, ok := c.groups[name]; ok {
		return g
	}

	group := groupcache.NewGroup(name, cacheBytes, groupcache.GetterFunc(
		func(c groupcache.Context, key string, dest groupcache.Sink) (err error) {
			if l, ok := c.(ctx.Logger); ok {
				l.Trace("cache.load", "Load data for key",
					log.String("group", name),
					log.String("key", key),
				)
			}

			var r []byte
			switch c := c.(type) {
			case context.Context:
				r, err = loader(c, key)
			default:
				r, err = loader(context.Background(), key)
			}
			if err != nil {
				return err
			}
			return dest.SetBytes(r)
		},
	))
	return &gcacheGroup{group}
}

// gcacheGroup wraps groupcache.Group and implement cache.Group
type gcacheGroup struct {
	g *groupcache.Group
}

func (g *gcacheGroup) Get(
	c context.Context, key string,
) (data []byte, err error) {
	if l, ok := c.(ctx.Logger); ok {
		l.Trace("cache.get", "Get data for key",
			log.String("group", g.g.Name()),
			log.String("key", key),
		)
	}

	err = g.g.Get(c, key, groupcache.AllocatingByteSliceSink(&data))
	return data, err
}

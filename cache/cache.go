// Package cache is a caching and cache-filling library
package cache

import (
	"github.com/stairlin/lego/ctx/journey"
	"github.com/stairlin/lego/disco"
)

type Cache interface {
	// NewGroup creates a LRU caching namespace with a size limit and a load
	// function to be called when the value is mising
	NewGroup(name string, cacheBytes int64, loader LoadFunc) Group
}

// A Group is a cache namespace
type Group interface {
	Get(ctx journey.Ctx, key string) ([]byte, error)
}

// A LoadFunc loads data for a key.
type LoadFunc func(context journey.Ctx, key string) ([]byte, error)

// Dependencies is an interface to "inject" required services
type Dependencies interface {
	Disco() disco.Agent
}

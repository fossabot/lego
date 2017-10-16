package journey

import (
	"sync"
)

// KV is a serialisable key/value store
// TODO: Replace it with go 1.9 sync/map at some point
type KV struct {
	mu sync.RWMutex

	Map map[interface{}]interface{}
}

func (a *KV) store(key interface{}, v interface{}) {
	a.mu.Lock()
	a.Map[key] = v
	a.mu.Unlock()
}

func (a *KV) load(key interface{}) interface{} {
	a.mu.RLock()
	v := a.Map[key]
	a.mu.RUnlock()
	return v
}

func (a *KV) delete(key interface{}) {
	a.mu.Lock()
	delete(a.Map, key)
	a.mu.Unlock()
}

func (a *KV) r(f func(key, value interface{}) bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	for key, value := range a.Map {
		if f(key, value) {
			return
		}
	}
}

func (a *KV) clone() *KV {
	copy := KV{}
	a.mu.RLock()
	for k, v := range a.Map {
		copy.Map[k] = v
	}
	a.mu.RUnlock()
	return &copy
}

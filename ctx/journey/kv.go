package journey

import (
	"sync"
)

// KV is an interface for key/value storage
type KV interface {
	Store(k string, v interface{})
	Retrieve(k string) (interface{}, bool)
	Delete(k string)
}

type kv struct {
	mu  sync.RWMutex
	Map map[string]interface{}
}

func newKV() *kv {
	return &kv{Map: map[string]interface{}{}}
}

func (a *kv) Store(k string, v interface{}) {
	a.mu.Lock()
	a.Map[k] = v
	a.mu.Unlock()
}

func (a *kv) Retrieve(k string) (interface{}, bool) {
	a.mu.RLock()
	v, ok := a.Map[k]
	a.mu.RUnlock()
	return v, ok
}

func (a *kv) Delete(k string) {
	a.mu.Lock()
	delete(a.Map, k)
	a.mu.Unlock()
}

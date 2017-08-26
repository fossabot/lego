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

// NewKV creates a new KV store
func NewKV() *KV {
	return &KV{Map: map[interface{}]interface{}{}}
}

// Store sets the value for a key. If the key already exists, it will
// be updated with the new value
func (a *KV) Store(key interface{}, v interface{}) {
	a.mu.Lock()
	a.Map[key] = v
	a.mu.Unlock()
}

// Load returns the value stored in the map for a key, or nil if no value is present.
// The ok result indicates whether value was found in the map.
func (a *KV) Load(key interface{}) (interface{}, bool) {
	a.mu.RLock()
	v, ok := a.Map[key]
	a.mu.RUnlock()
	return v, ok
}

// Delete deletes the value for a key. If the key does exist, it will be ignored
func (a *KV) Delete(key interface{}) {
	a.mu.Lock()
	delete(a.Map, key)
	a.mu.Unlock()
}

// Range calls f sequentially for each key and value present in the map.
// If f returns false, range stops the iteration.
func (a *KV) Range(f func(key, value interface{}) bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	for key, value := range a.Map {
		if f(key, value) {
			return
		}
	}
}

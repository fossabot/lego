package schedule

import "sync"

// peer is a peer node on a raft cluster
type peer struct {
	ID    string
	Addr  string
	Local bool
}

// peerMap is a peer map safe for concurrent use by multiple goroutines without
// additional locking or coordination. Loads, stores, and deletes run in
// amortised constant time.
type peerMap struct {
	m sync.Map
}

// Delete deletes the value for a key.
func (m *peerMap) Delete(key string) {
	m.m.Delete(key)
}

// Load returns the value stored in the map for a key, or nil if no value is present.
// The ok result indicates whether value was found in the map.
func (m *peerMap) Load(key string) (value *peer, ok bool) {
	v, ok := m.m.Load(key)
	if ok {
		return v.(*peer), true
	}
	return nil, false
}

// LoadOrStore returns the existing value for the key if present.
// Otherwise, it stores and returns the given value.
// The loaded result is true if the value was loaded, false if stored.
func (m *peerMap) LoadOrStore(key string, value *peer) (*peer, bool) {
	a, loaded := m.m.LoadOrStore(key, value)
	if loaded {
		return a.(*peer), true
	}
	return value, false
}

// Range calls f sequentially for each key and value present in the map.
// If f returns false, range stops the iteration.
func (m *peerMap) Range(f func(key string, value *peer) bool) {
	m.m.Range(func(key, value interface{}) bool {
		k := key.(string)
		v := value.(*peer)
		return f(k, v)
	})
}

// Store sets the value for a key.
func (m *peerMap) Store(key string, value *peer) {
	m.m.Store(key, value)
}

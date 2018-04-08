// Package lru implements a LRU cache.
//
// Source: https://github.com/vitessio/vitess/blob/master/go/cache/lru_cache.go
package lru

import (
	"container/list"
	"fmt"
	"sync"
	"time"
)

// Cache is a typical LRU cache implementation.  If the cache
// reaches the capacity, the least recently used item is deleted from
// the cache. Note the capacity is not the number of items, but the
// total sum of the Size() of each item.
type Cache struct {
	mu sync.Mutex

	// list & table contain *entry objects.
	list  *list.List
	table map[string]*list.Element

	size      int64
	capacity  int64
	evictions int64
}

// Value is the interface values that go into Cache need to satisfy
type Value interface {
	// Size returns how big this value is. If you want to just track
	// the cache by number of objects, you may return the size as 1.
	Size() int
}

// Item is what is stored in the cache
type Item struct {
	Key   string
	Value Value
}

type entry struct {
	key          string
	value        Value
	size         int64
	timeAccessed time.Time
}

// New creates a new empty cache with the given capacity.
func New(capacity int64) *Cache {
	return &Cache{
		list:     list.New(),
		table:    make(map[string]*list.Element),
		capacity: capacity,
	}
}

// Get returns a value from the cache, and marks the entry as most
// recently used.
func (lru *Cache) Get(key string) (v Value, ok bool) {
	lru.mu.Lock()
	defer lru.mu.Unlock()

	element := lru.table[key]
	if element == nil {
		return nil, false
	}
	lru.moveToFront(element)
	return element.Value.(*entry).value, true
}

// Peek returns a value from the cache without changing the LRU order.
func (lru *Cache) Peek(key string) (v Value, ok bool) {
	lru.mu.Lock()
	defer lru.mu.Unlock()

	element := lru.table[key]
	if element == nil {
		return nil, false
	}
	return element.Value.(*entry).value, true
}

// Set sets a value in the cache.
func (lru *Cache) Set(key string, value Value) {
	lru.mu.Lock()
	defer lru.mu.Unlock()

	if element := lru.table[key]; element != nil {
		lru.updateInplace(element, value)
	} else {
		lru.addNew(key, value)
	}
}

// SetIfAbsent will set the value in the cache if not present. If the
// value exists in the cache, we don't set it.
func (lru *Cache) SetIfAbsent(key string, value Value) {
	lru.mu.Lock()
	defer lru.mu.Unlock()

	if element := lru.table[key]; element != nil {
		lru.moveToFront(element)
	} else {
		lru.addNew(key, value)
	}
}

// Delete removes an entry from the cache, and returns if the entry existed.
func (lru *Cache) Delete(key string) bool {
	lru.mu.Lock()
	defer lru.mu.Unlock()

	element := lru.table[key]
	if element == nil {
		return false
	}

	lru.list.Remove(element)
	delete(lru.table, key)
	lru.size -= element.Value.(*entry).size
	return true
}

// Clear will clear the entire cache.
func (lru *Cache) Clear() {
	lru.mu.Lock()
	defer lru.mu.Unlock()

	lru.list.Init()
	lru.table = make(map[string]*list.Element)
	lru.size = 0
}

// SetCapacity will set the capacity of the cache. If the capacity is
// smaller, and the current cache size exceed that capacity, the cache
// will be shrank.
func (lru *Cache) SetCapacity(capacity int64) {
	lru.mu.Lock()
	defer lru.mu.Unlock()

	lru.capacity = capacity
	lru.checkCapacity()
}

// Stats returns a few stats on the cache.
func (lru *Cache) Stats() (length, size, capacity, evictions int64, oldest time.Time) {
	lru.mu.Lock()
	defer lru.mu.Unlock()
	if lastElem := lru.list.Back(); lastElem != nil {
		oldest = lastElem.Value.(*entry).timeAccessed
	}
	return int64(lru.list.Len()), lru.size, lru.capacity, lru.evictions, oldest
}

// StatsJSON returns stats as a JSON object in a string.
func (lru *Cache) StatsJSON() string {
	if lru == nil {
		return "{}"
	}
	l, s, c, e, o := lru.Stats()
	return fmt.Sprintf("{\"Length\": %v, \"Size\": %v, \"Capacity\": %v, \"Evictions\": %v, \"OldestAccess\": \"%v\"}", l, s, c, e, o)
}

// Length returns how many elements are in the cache
func (lru *Cache) Length() int64 {
	lru.mu.Lock()
	defer lru.mu.Unlock()
	return int64(lru.list.Len())
}

// Size returns the sum of the objects' Size() method.
func (lru *Cache) Size() int64 {
	lru.mu.Lock()
	defer lru.mu.Unlock()
	return lru.size
}

// Capacity returns the cache maximum capacity.
func (lru *Cache) Capacity() int64 {
	lru.mu.Lock()
	defer lru.mu.Unlock()
	return lru.capacity
}

// Evictions returns the eviction count.
func (lru *Cache) Evictions() int64 {
	lru.mu.Lock()
	defer lru.mu.Unlock()
	return lru.evictions
}

// Oldest returns the insertion time of the oldest element in the cache,
// or a IsZero() time if cache is empty.
func (lru *Cache) Oldest() (oldest time.Time) {
	lru.mu.Lock()
	defer lru.mu.Unlock()
	if lastElem := lru.list.Back(); lastElem != nil {
		oldest = lastElem.Value.(*entry).timeAccessed
	}
	return oldest
}

// Keys returns all the keys for the cache, ordered from most recently
// used to last recently used.
func (lru *Cache) Keys() []string {
	lru.mu.Lock()
	defer lru.mu.Unlock()

	keys := make([]string, 0, lru.list.Len())
	for e := lru.list.Front(); e != nil; e = e.Next() {
		keys = append(keys, e.Value.(*entry).key)
	}
	return keys
}

// Items returns all the values for the cache, ordered from most recently
// used to last recently used.
func (lru *Cache) Items() []Item {
	lru.mu.Lock()
	defer lru.mu.Unlock()

	items := make([]Item, 0, lru.list.Len())
	for e := lru.list.Front(); e != nil; e = e.Next() {
		v := e.Value.(*entry)
		items = append(items, Item{Key: v.key, Value: v.value})
	}
	return items
}

func (lru *Cache) updateInplace(element *list.Element, value Value) {
	valueSize := int64(value.Size())
	sizeDiff := valueSize - element.Value.(*entry).size
	element.Value.(*entry).value = value
	element.Value.(*entry).size = valueSize
	lru.size += sizeDiff
	lru.moveToFront(element)
	lru.checkCapacity()
}

func (lru *Cache) moveToFront(element *list.Element) {
	lru.list.MoveToFront(element)
	element.Value.(*entry).timeAccessed = time.Now()
}

func (lru *Cache) addNew(key string, value Value) {
	newEntry := &entry{key, value, int64(value.Size()), time.Now()}
	element := lru.list.PushFront(newEntry)
	lru.table[key] = element
	lru.size += newEntry.size
	lru.checkCapacity()
}

func (lru *Cache) checkCapacity() {
	// Partially duplicated from Delete
	for lru.size > lru.capacity {
		delElem := lru.list.Back()
		delValue := delElem.Value.(*entry)
		lru.list.Remove(delElem)
		delete(lru.table, delValue.key)
		lru.size -= delValue.size
		lru.evictions++
	}
}

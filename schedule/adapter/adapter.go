package adapter

import (
	"fmt"
	"sort"
	"sync"

	"github.com/stairlin/lego/config"
	"github.com/stairlin/lego/schedule"
	"github.com/stairlin/lego/schedule/adapter/local"
)

// Adapter returns a new agent initialised with the given config
type Adapter func(config.Tree) (schedule.Scheduler, error)

var (
	mu       sync.RWMutex
	adapters = make(map[string]Adapter)
)

func init() {
	// Register default adapters
	Register(local.Name, local.New)
}

// Adapters returns the list of registered adapters
func Adapters() []string {
	mu.RLock()
	defer mu.RUnlock()

	var l []string
	for a := range adapters {
		l = append(l, a)
	}

	sort.Strings(l)

	return l
}

// Register makes a scheduler adapter available by the provided name.
// If an adapter is registered twice or if an adapter is nil, it will panic.
func Register(name string, adapter Adapter) {
	mu.Lock()
	defer mu.Unlock()

	if adapter == nil {
		panic("schedule: Registered adapter is nil")
	}
	if _, dup := adapters[name]; dup {
		panic("schedule: Duplicated adapter")
	}

	adapters[name] = adapter
}

// New creates a new scheduler
func New(config config.Tree) (schedule.Scheduler, error) {
	mu.RLock()
	defer mu.RUnlock()

	keys := config.Keys()
	if len(keys) == 0 {
		return Null(), nil
	}
	adapter := keys[0]

	if f, ok := adapters[adapter]; ok {
		return f(config.Get(adapter))
	}
	return nil, fmt.Errorf("schedule adapter not found <%s>", adapter)
}

// Null returns a scheduler that does not do anything
func Null() schedule.Scheduler {
	return newNullScheduler()
}

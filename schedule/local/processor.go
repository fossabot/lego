package local

import (
	"container/heap"
	"sync"

	"github.com/pkg/errors"
	"github.com/stairlin/lego/schedule"
)

type processor struct {
	mu            sync.RWMutex
	registrations map[string]func(string, []byte) error
	q             queue
	stop          chan struct{}
}

func (p *processor) Start() {
	for {
		select {
		case <-p.stop:
			return
		}
	}
}

func (p *processor) Register(
	target string, fn func(string, []byte) error,
) (deregister func(), err error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, ok := p.registrations[target]; ok {
		return nil, errors.New("duplicate registration for target " + target)
	}

	p.registrations[target] = fn
	dereg := func() {
		delete(p.registrations, target)
	}
	return dereg, nil
}

func (p *processor) Close() error {
	p.stop <- struct{}{}
	return nil
}

// queue stores in-memory jobs that are about to be processed
type queue struct {
	mu sync.RWMutex
	L  []*event
}

type event struct {
	Job   *schedule.Job
	Due   int64 // priority in the queue (unix ns since epoch)
	Index int   // The Index of the event in the heap
}

func (q *queue) Len() int {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return len(q.L)
}

func (q *queue) Less(i, j int) bool {
	q.mu.RLock()
	defer q.mu.RUnlock()
	// the higher the number, the lower the priority
	return q.L[i].Due > q.L[j].Due
}

func (q *queue) Swap(i, j int) {
	q.mu.Lock()
	q.L[i], q.L[j] = q.L[j], q.L[i]
	q.L[i].Index = i
	q.L[j].Index = j
	q.mu.Unlock()
}

func (q *queue) Push(x interface{}) {
	event := x.(*event)
	event.Index = len(q.L)

	q.mu.Lock()
	q.L = append(q.L, event)
	q.mu.Unlock()

	heap.Fix(q, event.Index)
}

func (q *queue) Pop() interface{} {
	q.mu.Lock()
	defer q.mu.Unlock()

	n := len(q.L)
	event := q.L[n-1]
	event.Index = -1 // for safety
	q.L = q.L[0 : n-1]
	return event
}

func (q *queue) next() *schedule.Job {
	q.mu.RLock()
	defer q.mu.RUnlock()

	return q.L[len(q.L)-1].Job
}

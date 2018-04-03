// Package inmem implements an in-memory scheduler for testing purpose only
package inmem

import (
	"container/heap"
	"context"
	"errors"
	"math"
	"sync"
	"time"

	"github.com/stairlin/lego/schedule"
)

type scheduler struct {
	mu sync.RWMutex

	q             jobQueue
	registrations map[string]func(string, []byte) error
	stop          chan struct{}
	update        chan struct{}
}

// NewScheduler creates a new in-memory scheduler.
// It should only be used for testing purpose. JOBS ARE NOT PERSISTED.
func NewScheduler() schedule.Scheduler {
	s := &scheduler{
		registrations: make(map[string]func(string, []byte) error),
		stop:          make(chan struct{}),
		update:        make(chan struct{}, 1),
	}
	heap.Init(&s.q)
	return s
}

func (s *scheduler) Start() error {
	go s.dequeueEvents()
	return nil
}

func (s *scheduler) At(
	ctx context.Context,
	t time.Time,
	target string,
	data []byte,
	o ...schedule.JobOption,
) (string, error) {
	j := schedule.BuildJob(o...)
	j.Due = t.UnixNano()
	j.Target = target
	j.Data = data

	s.q.Push(&event{
		Job:     j,
		Attempt: 1,
		Due:     j.Due,
	})
	s.update <- struct{}{}

	return j.ID, nil
}

func (s *scheduler) In(
	ctx context.Context,
	d time.Duration,
	target string,
	data []byte,
	o ...schedule.JobOption,
) (string, error) {
	return s.At(ctx, time.Now().Add(d), target, data, o...)
}

func (s *scheduler) Register(
	target string, fn func(string, []byte) error,
) (deregister func(), err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.registrations[target]; ok {
		return nil, errors.New("duplicate registration for target " + target)
	}

	s.registrations[target] = fn
	dereg := func() {
		s.mu.Lock()
		delete(s.registrations, target)
		s.mu.Unlock()
	}
	return dereg, nil
}

func (s *scheduler) Close() error {
	s.stop <- struct{}{}
	return nil
}

func (s *scheduler) execute(i *event) {
	s.mu.RLock()
	fn, ok := s.registrations[i.Job.Target]
	s.mu.RUnlock()
	if !ok {
		// there is no handle for this target, so this job will be quietly discarded
		return
	}

	j := i.Job
	if err := fn(j.ID, j.Data); err == nil {
		return
	}

	lost := i.Attempt >= j.Options.RetryLimit
	stale := j.Options.AgeLimit != nil && time.Now().UnixNano()-j.Due > int64(*j.Options.AgeLimit)
	if lost || stale {
		return
	}

	// Push back to queue
	backoff := time.Second * time.Duration(math.Pow(2, float64(i.Attempt)))
	if backoff < j.Options.MinBackOff {
		backoff = j.Options.MinBackOff
	} else if backoff > j.Options.MaxBackOff {
		backoff = j.Options.MaxBackOff
	}
	s.q.Push(&event{
		Job:     j,
		Attempt: i.Attempt + 1,
		Due:     j.Due + int64(backoff),
	})
}

func (s *scheduler) dequeueEvents() {
	for {
		if s.q.Len() == 0 {
			select {
			case <-s.update:
				continue
			case <-s.stop:
				return
			}
		}

		d := s.q.next().Due - time.Now().UnixNano()
		if d <= 0 {
			// there is a slim chance to have a race condition, but both jobs would
			// have to be executed anyway
			s.execute(s.q.Pop().(*event))
			continue
		}

		select {
		case <-time.Tick(time.Duration(d)):
		case <-s.update:
		case <-s.stop:
			return
		}
	}
}

type jobQueue struct {
	mu sync.RWMutex
	L  []*event
}

func (q *jobQueue) Len() int {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return len(q.L)
}

func (q *jobQueue) Less(i, j int) bool {
	q.mu.RLock()
	defer q.mu.RUnlock()
	// the higher the number, the lower the priority
	return q.L[i].Due > q.L[j].Due
}

func (q *jobQueue) Swap(i, j int) {
	q.mu.Lock()
	q.L[i], q.L[j] = q.L[j], q.L[i]
	q.L[i].Index = i
	q.L[j].Index = j
	q.mu.Unlock()
}

func (q *jobQueue) Push(x interface{}) {
	q.mu.Lock()
	event := x.(*event)
	event.Index = len(q.L)

	q.L = append(q.L, event)
	q.mu.Unlock()

	// Fix calls other jobQueue functions, so it cannot be locked
	heap.Fix(q, event.Index)
}

func (q *jobQueue) Pop() interface{} {
	q.mu.Lock()
	defer q.mu.Unlock()

	n := len(q.L)
	event := q.L[n-1]
	event.Index = -1 // for safety
	q.L = q.L[0 : n-1]
	return event
}

func (q *jobQueue) next() *schedule.Job {
	q.mu.RLock()
	defer q.mu.RUnlock()

	return q.L[len(q.L)-1].Job
}

type event struct {
	Job     *schedule.Job
	Attempt uint32
	Due     int64 // priority in the queue (unix ns since epoch)
	Index   int   // The Index of the event in the heap
}

package schedule

import (
	"container/heap"
	"errors"
	"math"
	"sync"
	"time"
)

// NewInMem creates a new in-memory scheduler. It should only be used for testing
// purpose. JOBS ARE NOT PERSISTED.
func NewInMem() Scheduler {
	s := &inMemScheduler{
		registrations: make(map[string]func(string, []byte) error),
		stop:          make(chan struct{}),
		update:        make(chan struct{}, 1),
	}
	heap.Init(&s.q)
	return s
}

type inMemScheduler struct {
	mu sync.RWMutex

	q             jobQueue
	registrations map[string]func(string, []byte) error
	stop          chan struct{}
	update        chan struct{}
}

func (s *inMemScheduler) Start(config SchedulerConfig) error {
	go s.dequeueEvents()
	return nil
}

func (s *inMemScheduler) At(
	t time.Time, target string, data []byte, o ...JobOption,
) (string, error) {
	j := BuildJob()
	j.Due = t
	j.Target = target
	j.Data = data
	for _, o := range o {
		o(&j.Options)
	}

	s.q.Push(&event{
		Job: j,
		Due: j.Due.UnixNano(),
	})
	s.update <- struct{}{}

	return j.ID, nil
}

func (s *inMemScheduler) In(
	d time.Duration, target string, data []byte, o ...JobOption,
) (string, error) {
	return s.At(time.Now().Add(d), target, data, o...)
}

func (s *inMemScheduler) Register(
	target string, fn func(string, []byte) error,
) (deregister func(), err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.registrations[target]; ok {
		return nil, errors.New("duplicate registration for target " + target)
	}

	s.registrations[target] = fn
	dereg := func() {
		delete(s.registrations, target)
	}
	return dereg, nil
}

func (s *inMemScheduler) Close() error {
	s.stop <- struct{}{}
	return nil
}

func (s *inMemScheduler) execute(i *event) {
	s.mu.RLock()
	fn, ok := s.registrations[i.Job.Target]
	s.mu.RUnlock()
	if !ok {
		// there is no handle for this target, so this job will be quietly discarded
		return
	}

	j := i.Job
	j.Attempt++
	if err := fn(j.ID, j.Data); err == nil {
		return
	}

	if j.Attempt >= j.Options.RetryLimit ||
		(j.Options.AgeLimit != nil && time.Now().Sub(j.Due) > *j.Options.AgeLimit) {
		return
	}

	// Push back to queue
	backoff := time.Second * time.Duration(math.Pow(2, float64(j.Attempt)))
	if backoff < j.Options.MinBackOff {
		backoff = j.Options.MinBackOff
	} else if backoff > j.Options.MaxBackoff {
		backoff = j.Options.MaxBackoff
	}
	s.q.Push(&event{
		Job: j,
		Due: j.Due.UnixNano() + int64(backoff),
	})
}

func (s *inMemScheduler) dequeueEvents() {
	for {
		if s.q.Len() == 0 {
			select {
			case <-s.update:
				continue
			case <-s.stop:
				return
			}
		}

		d := s.q.next().Due.Sub(time.Now())
		if d <= 0 {
			// there is a slim chance to have a race condition, but both jobs would
			// have to be executed anyway
			s.execute(s.q.Pop().(*event))
			continue
		}

		select {
		case <-time.Tick(d):
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
	event := x.(*event)
	event.Index = len(q.L)

	q.mu.Lock()
	q.L = append(q.L, event)
	q.mu.Unlock()

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

func (q *jobQueue) next() *Job {
	q.mu.RLock()
	defer q.mu.RUnlock()

	return q.L[len(q.L)-1].Job
}

type event struct {
	Job   *Job
	Due   int64 // priority in the queue (unix ns since epoch)
	Index int   // The Index of the event in the heap
}

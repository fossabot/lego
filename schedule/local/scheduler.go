// Package local implements a scheduler that persists jobs on a local storage.
// The implementation currently uses BoltDB (https://github.com/boltdb/bolt).
package local

import (
	"math"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/stairlin/lego/schedule"
	pb "github.com/stairlin/lego/schedule/local/localpb"
)

// TODO:
// - Reap expired jobs (e.g. done, lost, or stale) (> 1 week - debugging)
// - Call subscriber in a go-routine
// - Create a pool of go-routines for subscribers
// - Update job (and the indices it impacts)

const (
	defaultUpdateBuffer = 16
)

type scheduler struct {
	mu sync.RWMutex

	config        Config
	registrations map[string]func(string, []byte) error
	storage       *storage

	updatec chan *pb.Job
	stopc   chan struct{}
}

// Config is the local scheduler configuration
type Config struct {
	// DB is the path to the database file
	DB string
	// InitialWindow defines how far in the past the initial load should go
	InitialWindow time.Duration
}

// NewScheduler creates a scheduler that persists data locally.
// This scheduler cannot be used on a distributed setup. Use net/schedule when
// running multiple lego instances.
func NewScheduler(c Config) schedule.Scheduler {
	if c.DB == "" {
		c.DB = "schedule.local.db"
	}
	if c.InitialWindow == 0 {
		c.InitialWindow = time.Second
	}
	return &scheduler{
		config:        c,
		registrations: make(map[string]func(string, []byte) error),
	}
}

// Open opens the storage
func (s *scheduler) Start() error {
	s.updatec = make(chan *pb.Job, defaultUpdateBuffer)
	s.stopc = make(chan struct{})

	s.storage = &storage{}
	if err := s.storage.Open(s.config.DB); err != nil {
		return err
	}

	last := s.storage.LastLoad()
	if last != 0 {
		s.config.InitialWindow = time.Duration(time.Now().UnixNano() - last + 1)
	}

	go s.watchJobs()
	return nil
}

func (s *scheduler) At(
	t time.Time, target string, data []byte, o ...schedule.JobOption,
) (string, error) {
	j := schedule.BuildJob(o...)
	j.Due = t.UnixNano()
	j.Target = target
	j.Data = data

	pbj := toPB(j)
	if err := s.storage.Save(pbj); err != nil {
		return "", err
	}
	s.notifyUpdate(pbj)
	return j.ID, nil
}

func (s *scheduler) In(
	d time.Duration, target string, data []byte, o ...schedule.JobOption,
) (string, error) {
	return s.At(time.Now().Add(d), target, data, o...)
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
		delete(s.registrations, target)
	}
	return dereg, nil
}

func (s *scheduler) Close() error {
	// TODO: Drain first
	close(s.stopc)
	s.storage.Close()
	close(s.updatec)
	return nil
}

func (s *scheduler) process(j *pb.Job) {
	s.mu.RLock()
	fn, ok := s.registrations[j.Target]
	s.mu.RUnlock()
	if !ok {
		// there is no handle for this target, so this job will be quietly discarded
		return
	}

	if err := fn(j.Id, j.Data); err != nil {
		s.failed(j, err)
		return
	}
	s.succeed(j)
}

func (s *scheduler) succeed(j *pb.Job) {
	// TODO: Mark job as updated
}

func (s *scheduler) failed(j *pb.Job, err error) {
	// TODO: Due is updated, so AgeLimit cannot work. Store initial due time

	now := time.Now().UnixNano()
	lost := j.Attempt >= j.Options.RetryLimit
	stale := j.Options.AgeLimit != -1 && now-j.Due > j.Options.AgeLimit
	if lost || stale {
		return
	}

	// Push back to storage
	backoff := int64(time.Second) * int64(math.Pow(2, float64(j.Attempt)))
	if backoff < j.Options.MinBackOff {
		backoff = j.Options.MinBackOff
	} else if backoff > j.Options.MaxBackOff {
		backoff = j.Options.MaxBackOff
	}
	j.Attempt++
	j.Due += int64(backoff)
	s.storage.Save(j)
	s.notifyUpdate(j)
}

func (s *scheduler) watchJobs() {
	from := time.Now().Add(-1 * s.config.InitialWindow).UnixNano()
	for {
		s.flushUpdates()

		to := time.Now().UnixNano()
		jobs, next, err := s.storage.Load(from, to)
		switch err {
		case nil:
		case errDatabaseClosed:
			return
		default:
			select {
			case <-time.Tick(time.Second):
				continue
			case <-s.stopc:
				return
			}
		}
		prev := from
		from = to + 1

		if len(jobs) == 0 {
			d := time.Duration(next - time.Now().UnixNano())
			if d <= 0 {
				continue
			}

			select {
			case <-time.Tick(d):
				continue
			case j := <-s.updatec:
				if prev <= j.Due && j.Due <= to {
					jobs = append(jobs, j)
				} else {
					continue
				}
			case <-s.stopc:
				return
			}
		}

		// TODO: Add to heap with `to`` (upper bound)
		for _, j := range jobs {
			s.process(j)
		}
	}
}

func (s *scheduler) notifyUpdate(j *pb.Job) {
	// TODO: Use context in case the request timed out
	select {
	case s.updatec <- j:
	case <-s.stopc:
	}
}

func (s *scheduler) flushUpdates() {
	for {
		select {
		case <-s.updatec:
		default:
			return
		}
	}
}

// toPB converts a schedule.Job to its protobuf counter part
func toPB(j *schedule.Job) *pb.Job {
	o := pb.JobOptions{
		RetryLimit: j.Options.RetryLimit,
		MinBackOff: int64(j.Options.MinBackOff),
		MaxBackOff: int64(j.Options.MaxBackOff),
	}
	if j.Options.AgeLimit != nil {
		o.AgeLimit = int64(*j.Options.AgeLimit)
	} else {
		o.AgeLimit = -1
	}

	return &pb.Job{
		Id:      j.ID,
		Target:  j.Target,
		Due:     j.Due,
		Data:    j.Data,
		Options: &o,
	}
}

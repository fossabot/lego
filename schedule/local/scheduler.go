// Package local implements a scheduler that persists jobs on a local storage.
// The implementation currently uses BoltDB (https://github.com/boltdb/bolt).
package local

import (
	"context"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/stairlin/lego/schedule"
	pb "github.com/stairlin/lego/schedule/local/localpb"
)

// TODO: Cleanup old events (Add window to config - e.g. keep 1 week for debugging purpose)

const (
	defaultDB           = "schedule.local.db"
	defaultUpdateBuffer = 16
	defaultWorkers      = 4
)

var (
	seededRand = rand.New(rand.NewSource(time.Now().UnixNano()))
	voidFn     = func(string, []byte) error { return nil }
)

type scheduler struct {
	mu sync.RWMutex

	config   Config
	handlers map[string]func(string, []byte) error

	// storage takes care of job/event/index persistence
	storage *storage
	// processor is a pool of goroutines that process events in parallel
	processor *processor
	// watcher watches for events to process
	watcher *watcher
}

// Config is the local scheduler configuration
type Config struct {
	// DB is the path to the database file
	DB string
	// Workers is the maximum number of goroutines that process jobs in parallel
	Workers int
	// Encryption activates data encryption.
	// It is worth noting that once a database created, it is no longer possible
	// to change this option.
	Encryption *EncryptionConfig
}

// EncryptionConfig is the configuration to encrypt data stored.
// The database encryption supports key rotation, so new keys can be added without
// affecting existing data. Old keys should be kept (almost) forever.
type EncryptionConfig struct {
	// Default is the key to use to encrypt new data
	Default uint32
	// Keys contains all encryption keys available
	Keys map[uint32][]byte
}

// NewScheduler creates a scheduler that persists data locally.
// This scheduler cannot be used on a distributed setup. Use net/schedule when
// running multiple lego instances.
func NewScheduler(c Config) schedule.Scheduler {
	if c.DB == "" {
		c.DB = defaultDB
	}
	if c.Workers == 0 {
		c.Workers = defaultWorkers
	}
	return &scheduler{
		config:   c,
		handlers: make(map[string]func(string, []byte) error),
	}
}

func (s *scheduler) Start() error {
	s.processor = newProcessor(s.config.Workers, s.process)
	s.storage = newStorage(s.config.Encryption)
	s.watcher = newWatcher(s.storage, s.processor.Exec())

	if err := s.storage.Open(s.config.DB); err != nil {
		return err
	}
	s.processor.Start()
	s.watcher.Start()
	return nil
}

func (s *scheduler) At(
	ctx context.Context,
	t time.Time,
	target string,
	data []byte,
	o ...schedule.JobOption,
) (string, error) {
	if target == "" {
		return "", errors.New("missing schedule target")
	}

	j := schedule.BuildJob(o...)
	j.Due = t.UnixNano()
	j.Target = target
	j.Data = data

	e := pb.Event{
		Due:     j.Due,
		Attempt: 1,
		Job:     toPB(j),
	}
	if err := s.storage.Save(&e); err != nil {
		return "", err
	}
	s.watcher.Notify(e.Due)
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

func (s *scheduler) HandleFunc(
	target string, fn func(string, []byte) error,
) (deregister func(), err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.handlers[target]; ok {
		return nil, errors.New("duplicate registration for target " + target)
	}

	s.handlers[target] = fn
	dereg := func() {
		delete(s.handlers, target)
	}
	return dereg, nil
}

func (s *scheduler) Drain() {
	s.watcher.Close()
	s.processor.Close()
}

func (s *scheduler) Close() error {
	s.storage.Close()
	return nil
}

func (s *scheduler) process(e *pb.Event) {
	j := e.Job

	expired := j.Options.AgeLimit != -1 &&
		time.Now().UnixNano() > j.Due+j.Options.AgeLimit
	if expired {
		return
	}

	fn := s.handler(j.Target)
	if err := fn(j.Id, j.Data); err == nil {
		// Job succeed
		return
	}

	// Job failed, prepare next attempt
	backoff := int64(time.Second) * int64(math.Pow(2, float64(e.Attempt)))
	if backoff < j.Options.MinBackOff {
		backoff = j.Options.MinBackOff
	} else if backoff > j.Options.MaxBackOff {
		backoff = j.Options.MaxBackOff
	}
	jitter := min(
		seededRand.Int63n(j.Options.MinBackOff*int64(e.Attempt)),
		j.Options.MaxBackOff,
	)

	next := pb.Event{
		Due:     e.Due + backoff + jitter,
		Attempt: e.Attempt + 1,
		Job:     e.Job,
	}

	if next.Attempt > j.Options.RetryLimit {
		return
	}
	if j.Options.AgeLimit != -1 && next.Due > j.Options.AgeLimit {
		return
	}

	s.storage.Save(&next)
	s.watcher.Notify(next.Due)
}

func (s *scheduler) handler(target string) func(string, []byte) error {
	s.mu.RLock()
	fn, ok := s.handlers[target]
	s.mu.RUnlock()
	if !ok {
		return voidFn
	}
	return fn
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

func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

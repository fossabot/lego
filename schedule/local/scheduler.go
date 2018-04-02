// Package local implements a scheduler that persists jobs on a local storage.
// The implementation currently uses BoltDB (https://github.com/boltdb/bolt).
package local

import (
	"time"

	"github.com/stairlin/lego/schedule"
	pb "github.com/stairlin/lego/schedule/local/localpb"
)

type scheduler struct {
	config    Config
	storage   *storage
	processor *processor
}

// Config is the local scheduler configuration
type Config struct {
	// DB is the path to the database file
	DB string
}

// NewScheduler creates a scheduler that persists data locally.
// This scheduler cannot be used on a distributed setup. Use net/schedule when
// running multiple lego instances.
func NewScheduler(c Config) schedule.Scheduler {
	if c.DB == "" {
		c.DB = "schedule.local.db"
	}
	return &scheduler{
		config: c,
	}
}

// Open opens the storage
func (s *scheduler) Start() error {
	s.storage = &storage{
		path: s.config.DB,
	}
	if err := s.storage.Open(); err != nil {
		return err
	}
	s.processor = &processor{
		registrations: make(map[string]func(string, []byte) error),
		stop:          make(chan struct{}),
	}

	go s.processor.Start()
	return nil
}

func (s *scheduler) At(
	t time.Time, target string, data []byte, o ...schedule.JobOption,
) (string, error) {
	j := schedule.BuildJob(o...)
	j.Due = t.UnixNano()
	j.Target = target
	j.Data = data

	if err := s.storage.Save(toPB(j)); err != nil {
		return "", err
	}
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
	return s.processor.Register(target, fn)
}

func (s *scheduler) Close() error {
	s.processor.Close()
	s.storage.Close()
	return nil
}

// toPB converts a schedule.Job to its protobuf counter part
func toPB(j *schedule.Job) *pb.Job {
	o := pb.JobOptions{
		RetryLimit:  j.Options.RetryLimit,
		MinBackOff:  int64(j.Options.MinBackOff),
		MaxBackOff:  int64(j.Options.MaxBackOff),
		Consistency: pb.Consistency(j.Options.Consistency),
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

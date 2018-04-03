package schedule

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/hashicorp/raft"
	"github.com/stairlin/lego/ctx/app"
	"github.com/stairlin/lego/log"
	"github.com/stairlin/lego/schedule"
)

// TODO:
// - Reap expired jobs (e.g. done, lost, or stale)
// - Job states
//   - Pending (attempt n)
//   - Running
//   - Succeed
//   - Failed (lost or stale)
//
//   Each job transition should be proposed to the raft cluster.
//
// - Consistency guarantee:
//     AtMostOnce:
//       When a job is stuck in running, because a failure occured, it should be
//       marked as failed, because we don't know whether it was executed or not
//     AtLeastOnce:
//       When a job is stuck in running, because a failure occured, it should be
//       re-run again on the same attempt, because we don't know whether it was executed or not

// scheduler is the raft state machine that coordinates job execution
type scheduler struct {
	mu  sync.RWMutex
	ctx app.Ctx
}

// Open opens the storage
func (s *scheduler) Start() error {
	return nil
}

func (s *scheduler) At(
	t time.Time, target string, data []byte, o ...schedule.JobOption,
) (string, error) {
	return "", nil
}

func (s *scheduler) In(
	d time.Duration, target string, data []byte, o ...schedule.JobOption,
) (string, error) {
	return s.At(time.Now().Add(d), target, data, o...)
}

func (s *scheduler) Register(
	target string, fn func(string, []byte) error,
) (deregister func(), err error) {
	return func() {}, nil
}

func (s *scheduler) Close() error {
	return nil
}

// Apply applies a Raft log entry to the storage
func (s *scheduler) Apply(l *raft.Log) interface{} {
	s.ctx.Trace("s.schedule.apply", "Apply log",
		log.Uint64("log_index", l.Index),
		log.Uint64("log_term", l.Term),
	)

	var c command
	if err := json.Unmarshal(l.Data, &c); err != nil {
		panic(fmt.Sprintf("failed to unmarshal command: %s", err.Error()))
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	switch c.Op {
	case "set":
		return s.applySet(c.Key, c.Value)
	case "delete":
		return s.applyDelete(c.Key)
	default:
		panic(fmt.Sprintf("unrecognized command op: %s", c.Op))
	}
}

// Snapshot returns a snapshot of the key-value store.
func (s *scheduler) Snapshot() (raft.FSMSnapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ctx.Trace("s.schedule.snapshot", "Snapshot data")

	return &snapshotter{}, nil
}

// Restore stores the key-value store to a previous state.
func (s *scheduler) Restore(rc io.ReadCloser) error {
	s.ctx.Trace("s.schedule.restore", "Restore data")

	// TODO: Restore Storage - Storage.Restore(rc)
	return rc.Close()
}

func (s *scheduler) applySet(key, value string) interface{} {

	return nil
}

func (s *scheduler) applyDelete(key string) interface{} {

	return nil
}

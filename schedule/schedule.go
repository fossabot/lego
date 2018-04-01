package schedule

import (
	"time"

	"github.com/google/uuid"
)

const (
	// DefaultRetryLimit is the default limit for retrying a failed job, measured
	// from when the job was first run.
	DefaultRetryLimit = 5
	// DefaultMinBackoff is the default minimum duration to wait before retrying
	// a job after it fails.
	DefaultMinBackoff = time.Second
	// DefaultMaxBackoff is the default maximum duration to wait before retrying
	// a job after it fails.
	DefaultMaxBackoff = time.Hour
)

// Scheduler is a time-based job scheduler. It executes jobs at fixed times or intervals
type Scheduler interface {
	// Start does the initialisation work to bootstrap a Scheduler. For example,
	// this function may start the event loop and watch the updates.
	Start(config SchedulerConfig) error

	// At registers a job that will be executed at time t
	At(t time.Time, target string, data []byte, o ...JobOption) (string, error)
	// In registers a job that will be executed in duration d from now
	In(d time.Duration, target string, data []byte, o ...JobOption) (string, error)
	// Cancel cancels a scheduled job.
	// When an id does not exist or is from an already-executed job, Cancel will ignore
	// the operation
	// Cancel(id string)
	// Register to all events from the given target. For each new job, fn will be
	// called.
	// There can be only one registration per target.
	Register(target string, fn func(id string, data []byte) error) (deregister func(), err error)

	// TODO: Interval API
	// It should create a Schedule struct that will generate Jobs (occurrences)
	// Interval(r RRule, target string, o ...JobOption) error

	// Close shuts down the scheduler.
	Close() error
}

type SchedulerConfig struct{}

// A Job is a one-time task executed at a specific time.
type Job struct {
	ID      string
	Target  string
	Due     time.Time
	Data    []byte
	Attempt uint32

	Options jobOptions
}

func BuildJob() *Job {
	return &Job{
		ID: uuid.New().String(),
		Options: jobOptions{
			RetryLimit: DefaultRetryLimit,
			MinBackOff: DefaultMinBackoff,
			MaxBackoff: DefaultMaxBackoff,
			Storage:    defaultStorageConfig,
		},
	}
}

// JobOption configures how we set up a job
type JobOption func(*jobOptions)

// jobOptions configure a Job. jobOptions are set by the JobOption values passed
// to At, In, or Interval.
type jobOptions struct {
	RetryLimit uint32
	MinBackOff time.Duration
	MaxBackoff time.Duration
	AgeLimit   *time.Duration

	Storage StorageConfig
}

// WithConsistency sets the job consistency guarantee when it uses a distributed scheduler
//
// It can either be executed at most once or at least once. The consistency guarantee
// strongly depends on the situation.
func WithConsistency(c Consistency) JobOption {
	return func(o *jobOptions) {
		o.Storage.Consistency = c
	}
}

// WithRetryLimit sets how many times a job can be retried upon failure.
//
// When omitted from the parameters, the limit is set to 'DefaultRetryLimit' by default.
func WithRetryLimit(l uint32) JobOption {
	return func(o *jobOptions) {
		o.RetryLimit = l
	}
}

// MinBackOff sets the minimum duration to wait before retrying a job after it fails.
func MinBackOff(d time.Duration) JobOption {
	return func(o *jobOptions) {
		o.MinBackOff = d
	}
}

// MaxBackoff sets the maximum duration to wait before retrying a job after it fails.
func MaxBackoff(d time.Duration) JobOption {
	return func(o *jobOptions) {
		o.MaxBackoff = d
	}
}

// WithAgeLimit sets a time limit for retrying a failed job, measured from when
// the job was first run. If specified with WithRetryLimit, the scheduler retries
// the job until both limits are reached.
func WithAgeLimit(d time.Duration) JobOption {
	return func(o *jobOptions) {
		o.AgeLimit = &d
	}
}

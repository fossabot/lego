package schedule

import (
	"io"
	"time"
)

type Storage interface {
	// Open opens the storage
	Open(config *StorageConfig) error
	// Close closes the storage
	Close() error

	// Load loads a cursor from the given moment in time
	Load(from time.Time) (Cursor, error)
	// Append appends a new job to the storage
	Append(j *Job) error

	// Snapshot backs up the data to w
	Snapshot(w io.Writer) error
	// Restore restores the data from r
	Restore(r io.Reader) error
}

type StorageConfig struct {
	Consistency Consistency
}

type Cursor interface {
	// Next blocks until there is a job to execute
	Next() *Job
	// Close closes the cursor
	Close() error
}

type Consistency uint8

const (
	// AtMostOnce is a consistency guarantee when a job is on a distributed scheduler
	// that ensures the job will be executed at most once.
	// That means it will be either executed once or not executed at all.
	AtMostOnce Consistency = iota
	// AtLeastOnce is a consistency guarantee when a job is on a distributed scheduler
	// that ensures the job will be executed at least once.
	// That means it will be either executed once or executed multiple times.
	AtLeastOnce
)

var defaultStorageConfig = StorageConfig{
	Consistency: AtMostOnce,
}

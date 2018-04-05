package cluster

import (
	"io"
	"path/filepath"
	"time"

	"github.com/boltdb/bolt"
	"github.com/pkg/errors"
	"github.com/stairlin/lego/ctx/app"
)

const (
	dbFileMod = 0600
)

var (
	jobBucket = []byte("job")
)

type storage struct {
	ctx app.Ctx

	db  *bolt.DB
	dir string
}

// Open opens the storage
func (s *storage) Open() error {
	db, err := bolt.Open(
		filepath.Join(s.dir, "schedule.db"),
		dbFileMod,
		&bolt.Options{Timeout: 1 * time.Second},
	)
	if err != nil {
		return errors.Wrap(err, "error opening schedule database")
	}

	s.db = db
	return nil
}

// Snapshot backs up the data to w
func (s *storage) Snapshot(w io.Writer) error {
	return nil
}

// Restore restores the data from r
func (s *storage) Restore(r io.Reader) error {
	return nil
}

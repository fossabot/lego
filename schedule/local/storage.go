package local

import (
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	pb "github.com/stairlin/lego/schedule/local/localpb"
)

const (
	dbFileMod   = 0600
	partitionBy = int64(time.Hour)
)

var (
	jobBucket   = []byte("job")
	jobIxBucket = []byte("job-index")
)

type storage struct {
	db   *bolt.DB
	path string
}

func (s *storage) Open() error {
	db, err := bolt.Open(
		s.path,
		dbFileMod,
		&bolt.Options{Timeout: 1 * time.Second},
	)
	if err != nil {
		return errors.Wrap(err, "error opening schedule database")
	}
	s.db = db
	return nil
}

// Save persists j to the local database and update the index
func (s *storage) Save(j *pb.Job) error {
	jobData, err := proto.Marshal(j)
	if err != nil {
		return errors.Wrap(err, "error marshalling job")
	}

	ixKey := indexKey(j)
	jobKey := jobKey(j)

	return s.db.Update(func(tx *bolt.Tx) error {
		jb, err := tx.CreateBucketIfNotExists(jobBucket)
		if err != nil {
			return errors.Wrap(err, "error creating/loading job bucket")
		}
		ib, err := tx.CreateBucketIfNotExists(jobIxBucket)
		if err != nil {
			return errors.Wrap(err, "error creating/loading index bucket")
		}

		// Load and add job to index
		ix := pb.Index{}
		if err := proto.Unmarshal(ib.Get(ixKey), &ix); err != nil {
			return errors.Wrap(err, "error unmarshalling index")
		}
		if j.Due < ix.Min {
			ix.Min = j.Due
		}
		if j.Due > ix.Max {
			ix.Max = j.Due
		}
		ix.Keys = append(ix.Keys, string(jobKey))
		sort.Strings(ix.Keys)
		ixData, err := proto.Marshal(&ix)
		if err != nil {
			return errors.Wrap(err, "error marshalling index")
		}

		// Persist job and index
		if err := jb.Put(jobKey, jobData); err != nil {
			return errors.Wrap(err, "error creating job record")
		}
		if err := ib.Put(ixKey, ixData); err != nil {
			return errors.Wrap(err, "error updating index record")
		}
		return nil
	})
}

func (s *storage) Close() error {
	if s.db == nil {
		return nil
	}
	return s.db.Close()
}

func indexKey(j *pb.Job) []byte {
	rem := j.Due % partitionBy
	return []byte(strconv.FormatInt(j.Due-rem, 10))
}

func jobKey(j *pb.Job) []byte {
	return []byte(strings.Join([]string{
		strconv.FormatInt(j.Due, 10),
		j.Id,
	}, "/"))
}

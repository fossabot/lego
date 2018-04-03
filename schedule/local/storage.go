package local

import (
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	pb "github.com/stairlin/lego/schedule/local/localpb"
)

// TODO: Implement encryption

const (
	dbFileMod   = 0600
	partitionBy = int64(time.Hour)
)

var (
	eventBucket     = []byte("event")
	partitionBucket = []byte("partition")
	logBucket       = []byte("log")

	bucketKeys = [][]byte{
		eventBucket,
		partitionBucket,
		logBucket,
	}

	errDatabaseClosed = errors.New("db closed")
)

type storage struct {
	state uint32
	db    *bolt.DB
}

func (s *storage) Open(path string) error {
	db, err := bolt.Open(
		path,
		dbFileMod,
		&bolt.Options{Timeout: 1 * time.Second},
	)
	if err != nil {
		return errors.Wrap(err, "error opening schedule database")
	}

	// Create buckets if needed
	err = db.Update(func(tx *bolt.Tx) error {
		for _, bk := range bucketKeys {
			_, err := tx.CreateBucketIfNotExists(bk)
			if err != nil {
				return errors.Wrapf(err, "error creating/loading bucket <%s>", bk)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	atomic.StoreUint32(&s.state, 1)
	s.db = db
	return nil
}

// Save persists j to the local database and update the index
func (s *storage) Save(e *pb.Event) error {
	if atomic.LoadUint32(&s.state) == 0 {
		return errDatabaseClosed
	}

	e.Id = e.Job.Id + "/" + strconv.FormatUint(uint64(e.Attempt), 10)
	evtData, err := proto.Marshal(e)
	if err != nil {
		return errors.Wrap(err, "error marshalling event")
	}

	partKey := partitionKey(e.Due)
	eventKey := eventKey(e)

	return s.db.Batch(func(tx *bolt.Tx) error {
		parts := tx.Bucket(partitionBucket)
		events := tx.Bucket(eventBucket)

		// Add event to partition
		part := pb.Partition{}
		partData := parts.Get(partKey)
		if len(partData) > 0 {
			if err := proto.Unmarshal(partData, &part); err != nil {
				return errors.Wrap(err, "error unmarshalling index")
			}
		}
		if part.From == 0 && part.To == 0 {
			part.From, part.To = partitionRange(e.Due)
		}
		part.Keys = append(part.Keys, string(eventKey))
		sort.Strings(part.Keys)
		partData, err := proto.Marshal(&part)
		if err != nil {
			return errors.Wrap(err, "error marshalling index")
		}

		if err := events.Put(eventKey, evtData); err != nil {
			return errors.Wrap(err, "error creating event record")
		}
		if err := parts.Put(partKey, partData); err != nil {
			return errors.Wrap(err, "error updating index record")
		}
		return nil
	})
}

func (s *storage) Load(from, to int64) (l []*pb.Event, next int64, err error) {
	if atomic.LoadUint32(&s.state) == 0 {
		return nil, 0, errDatabaseClosed
	}

	start, _ := partitionRange(from)
	end, _ := partitionRange(to)
	next = end + 1

	return l, next, s.db.Batch(func(tx *bolt.Tx) error {
		parts := tx.Bucket(partitionBucket)
		events := tx.Bucket(eventBucket)
		logs := tx.Bucket(logBucket)

		for t := start; t <= end; t += partitionBy {
			partKey := partitionKey(t)
			part := pb.Partition{}
			partData := parts.Get(partKey)
			if len(partData) > 0 {
				if err := proto.Unmarshal(partData, &part); err != nil {
					return errors.Wrap(err, "error unmarshalling index")
				}
			}

			for _, key := range part.Keys {
				e := pb.Event{}
				if err := proto.Unmarshal(events.Get([]byte(key)), &e); err != nil {
					return errors.Wrap(err, "error unmarshalling event")
				}
				if from <= e.Due && e.Due <= to {
					l = append(l, &e)
				}
				if to < e.Due && e.Due < next {
					next = e.Due
				}
			}
		}

		froms := strconv.FormatInt(from, 10)
		tos := strconv.FormatInt(to, 10)
		err := logs.Put(
			[]byte(tos),
			[]byte(froms+"-"+tos),
		)
		if err != nil {
			return errors.Wrap(err, "error creating log record")
		}
		return nil
	})
}

func (s *storage) LastLoad() (t int64) {
	s.db.Batch(func(tx *bolt.Tx) error {
		logs := tx.Bucket(logBucket)

		k, _ := logs.Cursor().Last()
		if len(k) > 0 {
			t, _ = strconv.ParseInt(string(k), 10, 64)
		}
		return nil
	})
	return t
}

func (s *storage) Close() error {
	if s.db == nil {
		return nil
	}
	atomic.StoreUint32(&s.state, 0)
	return s.db.Close()
}

func eventKey(e *pb.Event) []byte {
	return []byte(strings.Join([]string{
		strconv.FormatInt(e.Due, 10),
		e.Id,
	}, "/"))
}

func partitionKey(t int64) []byte {
	from, _ := partitionRange(t)
	return []byte(strconv.FormatInt(from, 10))
}

func partitionRange(t int64) (int64, int64) {
	from := t - (abs(t) % partitionBy)
	to := from + partitionBy - 1
	return from, to
}

func abs(i int64) int64 {
	if i < 0 {
		return i * -1
	}
	return i
}

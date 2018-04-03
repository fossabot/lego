package local

import (
	"fmt"
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
	eventBucket   = []byte("event")
	eventIxBucket = []byte("event-index")
	logBucket     = []byte("log")

	bucketKeys = [][]byte{
		eventBucket,
		eventIxBucket,
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

	evtData, err := proto.Marshal(e)
	if err != nil {
		return errors.Wrap(err, "error marshalling event")
	}

	ixKey := indexKey(e.Due)
	eventKey := eventKey(e)

	return s.db.Batch(func(tx *bolt.Tx) error {
		indices := tx.Bucket(eventIxBucket)
		events := tx.Bucket(eventBucket)

		// Load and add event to index
		ix := pb.Index{}
		ixData := indices.Get(ixKey)
		if len(ixData) > 0 {
			if err := proto.Unmarshal(ixData, &ix); err != nil {
				return errors.Wrap(err, "error unmarshalling index")
			}
		}
		if ix.From == 0 {
			ix.From = (e.Due - (abs(e.Due) % partitionBy))
		}
		if ix.To == 0 {
			ix.To = (e.Due - (abs(e.Due) % partitionBy) + partitionBy - 1)
		}
		ix.Keys = append(ix.Keys, string(eventKey))
		sort.Strings(ix.Keys)
		ixData, err := proto.Marshal(&ix)
		if err != nil {
			return errors.Wrap(err, "error marshalling index")
		}

		// Persist event and index
		if err := events.Put(eventKey, evtData); err != nil {
			return errors.Wrap(err, "error creating event record")
		}
		if err := indices.Put(ixKey, ixData); err != nil {
			return errors.Wrap(err, "error updating index record")
		}
		return nil
	})
}

func (s *storage) Load(from, to int64) (l []*pb.Event, next int64, err error) {
	if atomic.LoadUint32(&s.state) == 0 {
		return nil, 0, errDatabaseClosed
	}

	start := (from - (abs(from) % partitionBy))
	end := (to - (abs(to) % partitionBy))
	next = end + 1

	return l, next, s.db.Batch(func(tx *bolt.Tx) error {
		indices := tx.Bucket(eventIxBucket)
		events := tx.Bucket(eventBucket)
		logs := tx.Bucket(logBucket)

		for t := start; t <= end; t += partitionBy {
			ixKey := indexKey(t)
			ix := pb.Index{}
			ixData := indices.Get(ixKey)
			if len(ixData) > 0 {
				if err := proto.Unmarshal(ixData, &ix); err != nil {
					return errors.Wrap(err, "error unmarshalling index")
				}
			}

			for _, key := range ix.Keys {
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

		err := logs.Put(
			[]byte(strconv.FormatInt(to, 10)),
			[]byte(fmt.Sprintf("%d-%d", from, to)),
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

type partition struct {
	Min int64
	Max int64
	L   []*pb.Job
}

func (p *partition) InRange(i int64) bool {
	return p.Min <= i && i <= p.Max
}

func indexKey(t int64) []byte {
	rem := t % partitionBy
	return []byte(strconv.FormatInt(t-rem, 10))
}

func eventKey(e *pb.Event) []byte {
	return []byte(strings.Join([]string{
		strconv.FormatInt(e.Due, 10),
		e.Id,
	}, "/"))
}

func abs(i int64) int64 {
	if i < 0 {
		return i * -1
	}
	return i
}

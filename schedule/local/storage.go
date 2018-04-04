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
	"github.com/stairlin/lego/crypto"
	pb "github.com/stairlin/lego/schedule/local/localpb"
)

const (
	dbFileMod   = 0600
	partitionBy = int64(time.Hour)
)

var (
	// ErrMarshalling occurs when a storage message cannot be marshalled
	ErrMarshalling = errors.New("schedule marshalling error")
	// ErrUnmarshalling occurs when a storage message cannot be unmarshalled
	ErrUnmarshalling = errors.New("schedule unmarshalling error")

	eventBucket       = []byte("event")
	partitionBucket   = []byte("partition")
	checkpointBuckets = []byte("checkpoint")
	bucketKeys        = [][]byte{
		eventBucket,
		partitionBucket,
		checkpointBuckets,
	}

	lastCheckpointKey = []byte("last")

	errDatabaseClosed = errors.New("db closed")
)

type storage struct {
	state  uint32
	db     *bolt.DB
	crypto *crypto.Rotor
}

// Open opens the database. This function must be called first.
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

// Save persists e to the storage
func (s *storage) Save(e *pb.Event) error {
	if atomic.LoadUint32(&s.state) == 0 {
		return errDatabaseClosed
	}

	e.Id = e.Job.Id + "/" + strconv.FormatUint(uint64(e.Attempt), 10)
	evtData, err := s.marshal(e)
	if err != nil {
		return ErrMarshalling
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
			if err := s.unmarshal(partData, &part); err != nil {
				return ErrUnmarshalling
			}
		}
		if part.From == 0 && part.To == 0 {
			part.From, part.To = partitionRange(e.Due)
		}
		part.Keys = append(part.Keys, string(eventKey))
		sort.Strings(part.Keys)
		partData, err := s.marshal(&part)
		if err != nil {
			return ErrMarshalling
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

// Load loads events due within the given range and return the time on which
// the next event is due.
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
		checkpoints := tx.Bucket(checkpointBuckets)

		for t := start; t <= end; t += partitionBy {
			partKey := partitionKey(t)
			part := pb.Partition{}
			partData := parts.Get(partKey)
			if len(partData) > 0 {
				if err := s.unmarshal(partData, &part); err != nil {
					return ErrUnmarshalling
				}
			}

			for _, key := range part.Keys {
				e := pb.Event{}
				if err := s.unmarshal(events.Get([]byte(key)), &e); err != nil {
					return ErrUnmarshalling
				}
				if from <= e.Due && e.Due <= to {
					l = append(l, &e)
				}
				if to < e.Due && e.Due < next {
					next = e.Due
				}
			}
		}

		seq, err := checkpoints.NextSequence()
		if err != nil {
			return err
		}
		checkpointData, err := s.marshal(&pb.Checkpoint{
			Seq:  seq,
			From: from,
			To:   to,
		})
		if err != nil {
			return ErrMarshalling
		}
		if err := checkpoints.Put(lastCheckpointKey, checkpointData); err != nil {
			return errors.Wrap(err, "error creating log record")
		}
		return nil
	})
}

// LastCheckpoint returns the upper bound of the last load range.
func (s *storage) LastCheckpoint() (t int64) {
	// Default value to make sure old events won't be re-processed
	t = time.Now().UnixNano()

	s.db.Batch(func(tx *bolt.Tx) error {
		checkpoints := tx.Bucket(checkpointBuckets)

		data := checkpoints.Get(lastCheckpointKey)
		if len(data) == 0 {
			return nil
		}

		cp := pb.Checkpoint{}
		if err := s.unmarshal(data, &cp); err != nil {
			return err
		}
		if cp.Seq == checkpoints.Sequence() {
			t = cp.To
		}
		return nil
	})
	return t
}

// Close implements io.Closer
func (s *storage) Close() error {
	if s.db == nil {
		return nil
	}
	atomic.StoreUint32(&s.state, 0)
	return s.db.Close()
}

func (s *storage) marshal(pb proto.Message) ([]byte, error) {
	if s.crypto == nil {
		return proto.Marshal(pb)
	}

	plain, err := proto.Marshal(pb)
	if err != nil {
		return nil, err
	}
	return s.crypto.Encrypt(plain)
}

func (s *storage) unmarshal(buf []byte, pb proto.Message) error {
	if s.crypto == nil {
		return proto.Unmarshal(buf, pb)
	}

	plain, err := s.crypto.Decrypt(buf)
	if err != nil {
		return ErrUnmarshalling
	}
	return proto.Unmarshal(plain, pb)
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

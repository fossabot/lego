package cluster

import "github.com/hashicorp/raft"

type snapshotter struct{}

// Persist should dump all necessary state to the WriteCloser 'sink',
// and call sink.Close() when finished or call sink.Cancel() on error.
func (f *snapshotter) Persist(sink raft.SnapshotSink) error {
	err := func() error {
		// TODO: Get writer from Storage.Restore()
		var b []byte

		// Write data to sink.
		if _, err := sink.Write(b); err != nil {
			return err
		}

		// Close the sink.
		return sink.Close()
	}()

	if err != nil {
		sink.Cancel()
	}

	return err
}

// Release is invoked when we are finished with the snapshot.
func (f *snapshotter) Release() {}

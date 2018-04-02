package schedule

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
	"github.com/pkg/errors"
	"github.com/stairlin/lego/ctx/app"
	"github.com/stairlin/lego/disco"
	"github.com/stairlin/lego/log"
	lnet "github.com/stairlin/lego/net"
)

const (
	retainSnapshotCount = 3
	maxPool             = 3
	raftTimeout         = 10 * time.Second
)

// A Server defines parameters for running a lego compatible Schedule server
//
// Unlike lego/schedule, this package runs a distributed scheduler based on the
// Raft consensus.
type Server struct {
	mu sync.Mutex

	// # Server

	// id on the service discovery cluster
	id string
	// ctx is the app context on which the server is executed
	ctx app.Ctx
	// state holds the server state (up, down, or drain)
	state uint32
	// config contains the server configuration
	config serverConfig
	// done is called once the server has drained to stop the raft server
	done chan struct{}

	// # Raft
	leader          bool
	raft            *raft.Raft
	raftLogger      *logger
	store           *raftboltdb.BoltStore
	transport       *raft.NetworkTransport
	transportLogger *logger
	peers           peerMap

	// Scheduler
	scheduler *scheduler

	// # Service discovery

	// service is a link to service discovery that allows to run this server
	// in cluster mode
	service disco.Service
	// watcher listens to peer updates in order to propose config changes to the
	// cluster (e.g. add or remove node).
	watcher disco.Watcher
}

type serverConfig struct {
	raft raft.Config
	// dir is the directory where the current raft instance data is stored
	dir string
	// baseDir is the base directory where raft data is stored
	baseDir string
	// leaveGracefulTime is the graceful time given to a node before it starts
	// shutting down its raft instance
	leaveGracefulTime time.Duration
}

// NewServer creates a new schedule server with its default parameters
func NewServer(opts ...Option) *Server {
	s := &Server{
		config: serverConfig{
			raft:              *raft.DefaultConfig(),
			baseDir:           "schedule",
			leaveGracefulTime: 5 * time.Second,
		},
		done: make(chan struct{}),
	}

	s.AddOptions(opts...)
	return s
}

// Serve implements net.Server
func (s *Server) Serve(addr string, ctx app.Ctx) error {
	defer atomic.StoreUint32(&s.state, lnet.StateDown)
	s.ctx = ctx

	if s.id == "" {
		s.id = "local"
	}
	s.config.raft.LocalID = raft.ServerID(s.id)
	s.config.dir = filepath.Join(s.config.baseDir, s.id)

	// Wrap loggers
	s.raftLogger = newLogger(ctx, "raft")
	s.raftLogger.Start()
	s.transportLogger = newLogger(ctx, "transport")
	s.transportLogger.Start()

	s.config.raft.LogOutput = s.raftLogger

	// Create the snapshot store. This allows the Raft to truncate the log.
	snapshots, err := raft.NewFileSnapshotStore(
		s.config.dir, retainSnapshotCount, os.Stderr,
	)
	if err != nil {
		return fmt.Errorf("file snapshot store: %s", err)
	}

	// Create the log store and stable store.
	s.store, err = raftboltdb.NewBoltStore(filepath.Join(s.config.dir, "raft.db"))
	if err != nil {
		return fmt.Errorf("new bolt store: %s", err)
	}

	// Setup Raft communication.
	tcpaddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return err
	}
	s.transport, err = raft.NewTCPTransport(
		addr, tcpaddr, maxPool, raftTimeout, s.transportLogger,
	)
	if err != nil {
		return err
	}

	// Instantiate the Raft systems.
	s.raft, err = raft.NewRaft(
		&s.config.raft,
		s.scheduler,
		s.store,
		s.store,
		snapshots,
		s.transport,
	)
	if err != nil {
		return errors.Wrap(err, "error initialising raft")
	}

	// Bootstrap cluster
	if s.service == nil || len(s.service.Instances()) == 0 {
		s.ctx.Trace("s.schedule.bootstrap_clust", "Bootstrap cluster")

		configuration := raft.Configuration{
			Servers: []raft.Server{
				{
					ID:      s.config.raft.LocalID,
					Address: s.transport.LocalAddr(),
				},
			},
		}
		s.raft.BootstrapCluster(configuration)
	}

	go s.watchPeers()
	go s.watchLeadership()

	atomic.StoreUint32(&s.state, lnet.StateUp)
	<-s.done
	return nil
}

// Drain implements net.Server
func (s *Server) Drain() {
	s.mu.Lock()
	if atomic.LoadUint32(&s.state) != lnet.StateUp {
		s.mu.Unlock()
		return
	}
	atomic.StoreUint32(&s.state, lnet.StateDrain)
	s.mu.Unlock()

	isLeader := s.isLeader()
	if isLeader && s.numPeers() > 1 {
		fn := s.raft.RemoveServer(raft.ServerID(s.config.raft.LocalID), 0, 0)
		if err := fn.Error(); err != nil {
			s.ctx.Warning(
				"s.schedule.drain.leave.err",
				"Faild to remove leader from raft cluster",
				log.Error(err),
			)
		}
	}

	time.Sleep(s.config.leaveGracefulTime)

	if !isLeader {
		var left bool
		limit := time.Now().Add(5 * time.Second)
		for !left && time.Now().Before(limit) {
			// Sleep a while before we check.
			time.Sleep(50 * time.Millisecond)

			// Get the latest configuration.
			fn := s.raft.GetConfiguration()
			if err := fn.Error(); err != nil {
				s.ctx.Warning(
					"s.schedule.drain.check_err",
					"Failed to get raft configuration",
					log.Error(err),
				)
				break
			}

			// Check whether the node is still in the list
			left = true
			for _, server := range fn.Configuration().Servers {
				if string(server.ID) == s.id {
					left = false
					break
				}
			}
		}
	}

	s.ctx.Trace("s.schedule.drain.shutdown", "Node has left the cluster, shutting down")
	s.transport.Close()
	fn := s.raft.Shutdown()
	if err := fn.Error(); err != nil {
		s.ctx.Warning(
			"s.schedule.drain.shutdown_err",
			"Error shutting down raft",
			log.Error(err),
		)
	}
	s.store.Close()

	s.raftLogger.Stop()
	s.transportLogger.Stop()
	close(s.done)
}

// AddOptions applies opts to the server
func (s *Server) AddOptions(opts ...Option) {
	for _, opt := range opts {
		opt(s)
	}
}

// AddPeer proposes to add a node to the cluster, identified by id and
// located at addr.
// The node must be ready to respond to Raft communications at that address.
func (s *Server) AddPeer(id, addr string) error {
	s.ctx.Trace("s.schedule.peer.add", "Propose peer add",
		log.String("node_id", id),
		log.String("node_addr", addr),
	)

	s.peers.Store(id, &peer{
		ID:   id,
		Addr: addr,
	})
	f := s.raft.AddVoter(raft.ServerID(id), raft.ServerAddress(addr), 0, time.Second*3)
	if err := f.Error(); err != nil {
		return errors.Wrap(err, "error joining cluster")
	}
	return nil
}

// UpdatePeer proposes a peer update to the cluster
func (s *Server) UpdatePeer(id, addr string) error {
	s.ctx.Trace("s.schedule.peer.update", "Propose peer update",
		log.String("node_id", id),
		log.String("node_addr", addr),
	)

	s.peers.Store(id, &peer{
		ID:   id,
		Addr: addr,
	})
	f := s.raft.AddVoter(raft.ServerID(id), raft.ServerAddress(addr), 0, 0)
	if err := f.Error(); err != nil {
		return errors.Wrap(err, "error joining cluster")
	}
	return nil
}

// RemovePeer proposes a peer removal from the cluster
func (s *Server) RemovePeer(id string) error {
	s.ctx.Trace("s.schedule.peer.remove", "Propose peer removal",
		log.String("node_id", id),
	)

	fn := s.raft.RemoveServer(raft.ServerID(id), 0, 0)
	if err := fn.Error(); err != nil {
		return errors.Wrap(err, "error joining cluster")
	}
	s.peers.Delete(id)
	return nil
}

func (s *Server) watchLeadership() {
	for {
		select {
		case <-s.raft.LeaderCh():
			// Notify?
		case <-s.done:
		}
	}
}

// watchPeers listens to service discovery updates (e.g. added or removed nodes)
func (s *Server) watchPeers() {
	if s.service == nil {
		return
	}
	s.watcher = s.service.Watch()

	for {
		events, err := s.watcher.Next()
		switch err {
		case nil:
		case disco.ErrWatcherClosed:
			return
		default:
			s.ctx.Warning("s.schedule.watch.err", "Watcher returned an error",
				log.Error(err),
			)
		}

		if !s.isLeader() {
			continue
		}

		for _, evt := range events {
			if evt.Instance.Local {
				continue
			}

			var err error
			switch evt.Op {
			case disco.Add:
				err = s.AddPeer(evt.Instance.ID, evt.Instance.Addr())
			case disco.Update:
				err = s.UpdatePeer(evt.Instance.ID, evt.Instance.Addr())
			case disco.Delete:
				if !evt.Instance.Local {
					err = s.RemovePeer(evt.Instance.ID)
				}
			}

			switch err {
			case nil:
			default:
				s.ctx.Warning("s.schedule.propose_peer_update.err",
					"Failed to propose config change",
					log.String("instance_id", evt.Instance.ID),
					log.String("instance_name", evt.Instance.Name),
					log.String("instance_addr", evt.Instance.Addr()),
					log.Error(err),
				)
			}
		}
	}
}

func (s *Server) isLeader() bool {
	return s.raft.State() == raft.Leader
}

func (s *Server) numPeers() (n int) {
	fn := s.raft.GetConfiguration()
	if err := fn.Error(); err != nil {
		return 0
	}
	conf := fn.Configuration()

	for _, s := range conf.Servers {
		if s.Suffrage == raft.Voter {
			n++
		}
	}
	return n
}

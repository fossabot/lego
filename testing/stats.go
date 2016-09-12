package testing

import (
	"testing"

	"github.com/stairlin/lego/log"
	"github.com/stairlin/lego/stats"
)

// Stats is a simple Stats interface useful for tests
type Stats struct {
	t *testing.T
}

// NewStats creates a new stats
func NewStats(t *testing.T) stats.Stats {
	return &Stats{t: t}
}

func (s *Stats) Start()                   {}
func (s *Stats) Stop()                    {}
func (s *Stats) Add(metric *stats.Metric) {}
func (s *Stats) SetLogger(l log.Logger)   {}

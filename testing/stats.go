package testing

import (
	"testing"
	"time"

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

func (s *Stats) Start()                                                         {}
func (s *Stats) Stop()                                                          {}
func (s *Stats) Add(metric *stats.Metric)                                       {}
func (s *Stats) SetLogger(l log.Logger)                                         {}
func (s *Stats) Count(key string, n interface{}, meta ...map[string]string)     {}
func (s *Stats) Inc(key string, meta ...map[string]string)                      {}
func (s *Stats) Dec(key string, meta ...map[string]string)                      {}
func (s *Stats) Gauge(key string, n interface{}, meta ...map[string]string)     {}
func (s *Stats) Timing(key string, d time.Duration, meta ...map[string]string)  {}
func (s *Stats) Histogram(key string, n interface{}, tags ...map[string]string) {}

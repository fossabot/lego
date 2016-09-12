package stats

import (
	"time"

	"github.com/stairlin/lego/log"
)

// Stats is an interface for app statistics
type Stats interface {
	Start()
	Stop()
	Add(metric *Metric)

	SetLogger(l log.Logger)
}

// Metric is a measure at a given time
type Metric struct {
	Key    string
	Values map[string]interface{}
	T      time.Time
	Meta   map[string]string
}

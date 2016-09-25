package stats

import (
	"time"

	"github.com/stairlin/lego/log"
)

// Stats is an interface for app statistics
type Stats interface {
	Start()
	Stop()

	Count(key string, n interface{}, meta ...map[string]string)
	Inc(key string, meta ...map[string]string)
	Dec(key string, meta ...map[string]string)
	Gauge(key string, n interface{}, meta ...map[string]string)
	Timing(key string, t time.Duration, meta ...map[string]string)
	Histogram(key string, n interface{}, tags ...map[string]string)

	SetLogger(l log.Logger)
}

// Metric is a measure at a given time
type Metric struct {
	Key    string
	Values map[string]interface{}
	T      time.Time
	Meta   map[string]string
}

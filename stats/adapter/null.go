package adapter

import (
	"time"

	"github.com/stairlin/lego/log"
)

// Null is a stats adapter that does not do anything
type null struct{}

func (s *null) Start()                                                         {}
func (s *null) Stop()                                                          {}
func (s *null) SetLogger(l log.Logger)                                         {}
func (s *null) Count(key string, n interface{}, meta ...map[string]string)     {}
func (s *null) Inc(key string, meta ...map[string]string)                      {}
func (s *null) Dec(key string, meta ...map[string]string)                      {}
func (s *null) Gauge(key string, n interface{}, meta ...map[string]string)     {}
func (s *null) Timing(key string, t time.Duration, meta ...map[string]string)  {}
func (s *null) Histogram(key string, n interface{}, tags ...map[string]string) {}

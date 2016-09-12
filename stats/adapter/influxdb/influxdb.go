package influxdb

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/stairlin/lego/config"
	"github.com/stairlin/lego/log"
	"github.com/stairlin/lego/stats"
	influx "github.com/influxdata/influxdb/client/v2"
)

// Name contains the adapter registered name
const Name = "influxdb"

const (
	nanoseconds timeUnit = "ns"
	seconds              = "s"
	hours                = "h"
	days                 = "d"
	weeks                = "w"
	infinite             = "INF"
)

type timeUnit string

type Config struct {
	Type      string
	URI       string
	Username  string
	Password  string
	Database  string
	Precision string
}

func New(c map[string]string) (stats.Stats, error) {
	// Make client
	url := config.ValueOf(c["url"])
	client, err := influx.NewHTTPClient(influx.HTTPConfig{
		Addr:     url,
		Username: c["username"],
		Password: c["password"],
	})
	if err != nil {
		return nil, fmt.Errorf("influxdb client err (%s)", err)
	}

	// Create a new point batch
	dbConfig := influx.BatchPointsConfig{
		Database:  c["database"],
		Precision: c["precision"],
	}

	return &InfluxDB{
		url:    url,
		client: client,
		config: dbConfig,
		done:   make(chan bool, 1),
	}, nil
}

// InfluxDB is a stats module that sends data to... *drumroll*... InfluxDB!
type InfluxDB struct {
	mu      sync.Mutex
	metrics []*stats.Metric
	url     string
	client  influx.Client
	config  influx.BatchPointsConfig
	logger  log.Logger
	done    chan bool
}

func (s *InfluxDB) SetLogger(l log.Logger) {
	s.logger = l
}

func (s *InfluxDB) Add(metric *stats.Metric) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.metrics = append(s.metrics, metric)
}

func (s *InfluxDB) Start() {
	s.logger.Infof("stats influxdb: connecting to <%s>", s.url)

	tick := time.Tick(5 * time.Second)
	for {
		select {
		case <-s.done:
			break
		case <-tick:
			batch, err := s.buildBatch()
			if err != nil {
				s.logger.Errorf("cannot build influxdb batch <%s>", err)
			}

			if len(batch.Points()) == 0 {
				continue // don't send an empty batch
			}

			err = s.client.Write(batch)
			if err != nil {
				if strings.Contains(err.Error(), "database not found") {
					s.logger.Warningf("database <%s> does not exist", s.config.Database)
					s.createDB()
				} else {
					s.logger.Errorf("cannot write influxdb batch <%s>", err)
				}
			}
		}
	}
}

func (s *InfluxDB) Stop() {
	s.done <- true
	s.client.Close()
	s.logger.Info("Stopping InfluxDB stats client")
}

func (s *InfluxDB) buildBatch() (influx.BatchPoints, error) {
	batch, err := influx.NewBatchPoints(s.config)
	if err != nil {
		return nil, err
	}

	// Flush metrics
	s.mu.Lock()
	metrics := s.metrics
	s.metrics = []*stats.Metric{}
	s.mu.Unlock()

	// Build points
	for _, metric := range metrics {
		tags := metric.Meta
		fields := metric.Values

		p, err := influx.NewPoint(
			metric.Key,
			tags,
			fields,
			metric.T,
		)
		if err != nil {
			s.logger.Errorf("cannot build influxdb point <%s>", err)
			continue
		}

		batch.AddPoint(p)
	}

	return batch, nil
}

func (s *InfluxDB) createDB() {
	q := influx.NewQuery(
		fmt.Sprintf("CREATE DATABASE %s", s.config.Database),
		"",
		"",
	)
	if response, err := s.client.Query(q); err == nil && response.Error() != nil {
		s.logger.Errorf("cannot create database <%s> (%s)", s.config.Database, err)
	}
}

package config

import (
	"time"
)

// Config defines the app config
type Config struct {
	Node    string      `json:"node"`
	Version string      `json:"version"`
	Request Request     `json:"request"`
	Log     Log         `json:"log"`
	Stats   Stats       `json:"stats"`
	App     interface{} `json:"app"`
}

// Log contains all log-related configuration
type Log struct {
	Level     string        `json:"level"`
	Formatter AdapterConfig `json:"formatter"`
	Printer   AdapterConfig `json:"printer"`
}

// AdapterConfig is a generic config struct for adapters
type AdapterConfig struct {
	Adapter string            `json:"adapter"`
	Config  map[string]string `json:"config"`
}

// Stats contains all stats-related configuration
type Stats struct {
	On      bool              `json:"on"`
	Adapter string            `json:"adapter"`
	Config  map[string]string `json:"config"`
}

// Request defines the request default configuration
type Request struct {
	TimeoutMS    time.Duration `json:"timeout_ms"`
	AllowContext bool          `json:"allow_context"`
	Panic        bool          `json:"panic"`
}

// Timeout returns the TimeoutMS field in time.Duration
func (r *Request) Timeout() time.Duration {
	return time.Millisecond * r.TimeoutMS
}

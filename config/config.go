package config

import "time"

// TODO: Move these structs to lego package once ctx.App is gone

// Config defines the app config
type Config struct {
	Node    string  `toml:"node"`
	Version string  `toml:"version"`
	Request Request `toml:"request"`
}

// Request defines the request default configuration
type Request struct {
	TimeoutMS    time.Duration `toml:"timeout_ms"`
	AllowContext bool          `toml:"allow_context"`
	Panic        bool          `toml:"panic"`
}

// Timeout returns the TimeoutMS field in time.Duration
func (r *Request) Timeout() time.Duration {
	return time.Millisecond * r.TimeoutMS
}

package config

// Config defines the app config
type Config struct {
	Service string      `json:"service"`
	Node    string      `json:"node"`
	Version string      `json:"version"`
	Log     Log         `json:"log"`
	Stats   Stats       `json:"stats"`
	App     interface{} `json:"app"`
}

// Log contains all log-related configuration
type Log struct {
	Level     string `json:"level"`
	Formatter struct {
		Adapter string            `json:"adapter"`
		Config  map[string]string `json:"config"`
	} `json:"formatter"`
	Printer struct {
		Adapter string            `json:"adapter"`
		Config  map[string]string `json:"config"`
	} `json:"printer"`
}

// Stats contains all stats-related configuration
type Stats struct {
	On      bool              `json:"on"`
	Adapter string            `json:"adapter"`
	Config  map[string]string `json:"config"`
}

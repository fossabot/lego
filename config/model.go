package config

// Config defines the app config
type Config struct {
	Log   Log         `json:"log"`
	Stats Stats       `json:"stats"`
	App   interface{} `json:"app"`
}

// Log contains all log-related configuration
type Log struct {
	Level  string            `json:"level"`
	Output string            `json:"output"`
	Config map[string]string `json:"config"`
}

// Stats contains all stats-related configuration
type Stats struct {
	On      bool              `json:"on"`
	Adapter string            `json:"adapter"`
	Config  map[string]string `json:"config"`
}

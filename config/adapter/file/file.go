// Package file reads configuration from a JSON file
package file

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"

	a "github.com/stairlin/lego/config/adapter"
)

// Name contains the adapter registered name
const Name = "file"

// New returns a new file config store
func New(uri *url.URL) (a.Store, error) {
	if _, err := os.Stat(uri.Path); os.IsNotExist(err) {
		return nil, fmt.Errorf("file does not exist (%s) - %s", uri, err)
	}

	return &Store{Path: uri.Path}, nil
}

// Store reads config from a JSON file
type Store struct {
	Path string
}

// Load config for the given environment
func (s *Store) Load(config interface{}) error {
	// Load file
	file, err := os.Open(s.Path)
	if err != nil {
		return fmt.Errorf("config file cannot be opened (%s)", err)
	}

	// Read file
	r, err := ioutil.ReadAll(file)
	if err != nil {
		return fmt.Errorf("config file cannot be read (%s)", err)
	}

	// Unmarshal
	json.Unmarshal(r, config)

	return nil
}

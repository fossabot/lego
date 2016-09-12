// Package etcd reads configuration from etcd
package etcd

import (
	"fmt"
	"net/url"
	"time"

	"github.com/coreos/etcd/client"
	"golang.org/x/net/context"

	a "github.com/stairlin/lego/config/adapter"
)

// Name contains the adapter registered name
const Name = "etcd"

// New returns a new etcd config adapter
func New(uri *url.URL) (a.Store, error) {
	// Create etcd config
	cfg := client.Config{
		Endpoints: []string{uri.String()},
		Transport: client.DefaultTransport,
		// set timeout per request to fail fast when the target endpoint is unavailable
		HeaderTimeoutPerRequest: time.Second,
		Username:                uri.User.Username(),
	}

	if pwd, ok := uri.User.Password(); ok {
		cfg.Password = pwd
	}

	// Build etcd client
	etcd, err := client.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("etcd error (%s)", err)
	}

	return &Store{etcd: etcd}, nil
}

// Store reads config from etcd
type Store struct {
	etcd client.Client
}

// Load config for the given environment
func (s *Store) Load(config interface{}) error {
	kapi := client.NewKeysAPI(s.etcd)

	// Ẇrite data
	_, err := kapi.Set(context.Background(), "/foo", "bar", nil)
	if err != nil {
		return err
	}

	// Read data
	_, err = kapi.Get(context.Background(), "/foo", nil)
	if err != nil {
		return err
	}

	return nil
}

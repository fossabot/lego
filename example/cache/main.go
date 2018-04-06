// Package main is a distributed-cache example
//
// It creates a cache server and register it to the service discovery agent
package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/stairlin/lego"
	"github.com/stairlin/lego/cache"
	"github.com/stairlin/lego/log"
	netCache "github.com/stairlin/lego/net/cache"
	"github.com/stairlin/lego/net/naming"
)

type AppConfig struct {
	Foo string `json:"foo"`
}

var (
	seededRand = rand.New(rand.NewSource(time.Now().UnixNano()))
	charset    = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
)

func main() {
	// Create lego
	config := &AppConfig{}
	app, err := lego.New("api", config)
	if err != nil {
		fmt.Println("Problem initialising lego", err)
		os.Exit(1)
	}

	port, err := strconv.Atoi(os.Getenv("CACHE_PORT"))
	if err != nil {
		fmt.Println("Problem parsing port", err)
		os.Exit(1)
	}
	tags := []string{"v1"}

	// Register cache service
	cacheServer := netCache.NewServer(app.Ctx().Cache())
	app.Ctx().SetCache(cacheServer)
	app.RegisterService(&lego.ServiceRegistration{
		Name:   "api.cache",
		Host:   "127.0.0.1",
		Port:   uint16(port),
		Server: cacheServer,
		Tags:   tags,
	})

	// Listen to service updates
	w, err := naming.Resolve(app.Ctx(), "disco://api.cache?tag=v1")
	if err != nil {
		fmt.Println("Problem building watcher", err)
		os.Exit(1)
	}
	cacheServer.SetOptions(netCache.OptWatcher(w))

	// Cache random data
	grp := cacheServer.NewGroup("foo", 64<<20, cache.LoadFunc(
		func(ctx context.Context, key string) ([]byte, error) {
			return []byte(app.Config().Node), nil
		},
	))
	go func() {
		for {
			if app.Ctx().Err() != nil {
				return
			}
			key := genRandomString()
			v, err := grp.Get(app.Ctx(), key)
			if err != nil {
				app.Ctx().Error("example.cache.err", "Error loading data",
					log.Error(err),
				)
				continue
			}

			fmt.Println(app.Config().Node, "got key", key, "from node", string(v))
			time.Sleep(time.Second * time.Duration(rand.Int63n(15)))
		}
	}()

	// Start serving requests
	err = app.Serve()
	if err != nil {
		fmt.Println("Problem serving requests", err)
		os.Exit(1)
	}
}

func genRandomString() string {
	b := make([]byte, 1)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

// Package main is a distributed-cache example
//
// It creates a cache server and register it to the service discovery agent
package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"time"

	"github.com/stairlin/lego"
	"github.com/stairlin/lego/cache"
	"github.com/stairlin/lego/ctx/app"
	"github.com/stairlin/lego/ctx/journey"
	"github.com/stairlin/lego/log"
	"github.com/stairlin/lego/net/http"
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

	// Cache random data
	grp := app.Cache().NewGroup("foo", 64<<20, cache.LoadFunc(
		func(ctx journey.Ctx, key string) ([]byte, error) {
			ctx.Warning("cache.load", "Filling cache...", log.String("key", key))
			return []byte(app.Config().Node), nil
		},
	))
	go func() {
		for {
			if app.Ctx().Err() != nil {
				return
			}
			key := genRandomString()
			ctx := journey.New(app.Ctx())
			v, err := grp.Get(ctx, key)
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

	// Register HTTP handler
	h := handler{
		ctx:   app.Ctx(),
		cache: grp,
	}
	s := http.NewServer()
	s.HandleFunc("/cache/{key}", http.GET, h.Load)
	app.RegisterServer("127.0.0.1:3000", s)

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

type handler struct {
	ctx   app.Ctx
	cache cache.Group
}

// Cache handler example
func (h *handler) Load(ctx journey.Ctx, w http.ResponseWriter, r *http.Request) {
	ctx.Trace("http.cache.load", "Load data", log.String("key", r.Params["key"]))
	v, err := h.cache.Get(ctx, r.Params["key"])
	if err != nil {
		ctx.Warning("http.cache.err", "Error pulling data from cache", log.Error(err))
		w.Head(http.StatusInternalServerError)
		return
	}
	w.Data(http.StatusOK, "text/plain", ioutil.NopCloser(bytes.NewReader(v)))
}

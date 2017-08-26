package journey_test

import (
	"bytes"
	"encoding/gob"
	"reflect"
	"testing"

	"github.com/stairlin/lego/ctx/journey"
	lt "github.com/stairlin/lego/testing"
)

// TestVerbatim ensures that data stored can be retrieved
func TestVerbatim(t *testing.T) {
	tt := lt.New(t)
	j := journey.New(tt.NewAppCtx("journey-test"))

	k := "foo"
	v := "bar"
	j.KV().Store(k, v)

	res, ok := j.KV().Load(k)
	if !ok {
		t.Fatal("expect to load value")
	}
	if res != v {
		t.Errorf("expect to get %s, but got %s", v, res)
	}

	j.KV().Delete(k)

	res, ok = j.KV().Load(k)
	if ok {
		t.Error("expect to NOT load value")
	}
	if res == v {
		t.Error("expect value to have been deleted")
	}
}

func TestKV_Marshal(t *testing.T) {
	kv := journey.NewKV()
	kv.Store("lang", "en")
	kv.Store("uuid", "568c9247-8cdc-433c-b6f7-74e881665223")
	kv.Store("ip", "0.15.50.199")
	kv.Store("groups", []string{"alpha", "beta"})

	expect := *kv            // Copy KV store
	var network bytes.Buffer // Stand-in for the network.

	// Create an encoder and send a value.
	enc := gob.NewEncoder(&network)
	if err := enc.Encode(kv); err != nil {
		t.Fatal("encode:", err)
	}

	// Create a decoder and receive a value.
	kv = &journey.KV{}
	dec := gob.NewDecoder(&network)
	if err := dec.Decode(kv); err != nil {
		t.Fatal("encode:", err)
	}

	if !reflect.DeepEqual(expect.Map, kv.Map) {
		t.Errorf("expect map %v, but got %v", expect.Map, kv.Map)
	}
}

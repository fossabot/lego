package journey_test

import (
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

	res, ok := j.KV().Retrieve(k)
	if !ok {
		t.Fatal("expect to retrieve value")
	}
	if res != v {
		t.Errorf("expect to get %s, but got %s", v, res)
	}

	j.KV().Delete(k)

	res, ok = j.KV().Retrieve(k)
	if ok {
		t.Error("expect to NOT retrieve value")
	}
	if res == v {
		t.Error("expect value to have been deleted")
	}
}

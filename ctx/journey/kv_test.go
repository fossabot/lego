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
	j.Store(k, v)

	res := j.Load(k)
	if res != v {
		t.Errorf("expect to get %s, but got %s", v, res)
	}

	j.Delete(k)

	res = j.Load(k)
	if res == v {
		t.Error("expect value to have been deleted")
	}
}

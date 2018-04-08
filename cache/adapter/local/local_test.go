package local_test

import (
	"testing"

	"github.com/stairlin/lego/cache/adapter/local"
	"github.com/stairlin/lego/config"
	"github.com/stairlin/lego/ctx/journey"
	lt "github.com/stairlin/lego/testing"
)

func TestCache(t *testing.T) {
	tt := lt.New(t)
	app := tt.NewAppCtx("journey-test")

	cache, err := local.New(config.NullTree(), app)
	if err != nil {
		t.Fatal(err)
	}

	// Create group which can hold 2 keys
	expect := []byte("bar")
	var load int
	group := cache.NewGroup("foo", 6, func(ctx journey.Ctx, key string) ([]byte, error) {
		load++
		return expect, nil
	})

	// Store first
	ctx := journey.New(app)
	got, err := group.Get(ctx, "alpha")
	if err != nil {
		t.Fatal(err)
	}
	if string(expect) != string(got) {
		t.Errorf("expect to get %s, but got %s", string(expect), string(got))
	}
	got, err = group.Get(ctx, "alpha")
	if err != nil {
		t.Fatal(err)
	}
	if string(expect) != string(got) {
		t.Errorf("expect to get %s, but got %s", string(expect), string(got))
	}
	if load != 1 {
		t.Errorf("Expect to load data once, but got %d", load)
	}

	// Store second
	got, err = group.Get(ctx, "beta")
	if err != nil {
		t.Fatal(err)
	}
	if string(expect) != string(got) {
		t.Errorf("expect to get %s, but got %s", string(expect), string(got))
	}
	got, err = group.Get(ctx, "beta")
	if err != nil {
		t.Fatal(err)
	}
	if string(expect) != string(got) {
		t.Errorf("expect to get %s, but got %s", string(expect), string(got))
	}
	if load != 2 {
		t.Errorf("Expect to load data once, but got %d", load)
	}

	// Store third
	got, err = group.Get(ctx, "gamma")
	if err != nil {
		t.Fatal(err)
	}
	if string(expect) != string(got) {
		t.Errorf("expect to get %s, but got %s", string(expect), string(got))
	}
	got, err = group.Get(ctx, "gamma")
	if err != nil {
		t.Fatal(err)
	}
	if string(expect) != string(got) {
		t.Errorf("expect to get %s, but got %s", string(expect), string(got))
	}
	if load != 3 {
		t.Errorf("Expect to load data once, but got %d", load)
	}

	// Ensure the second is still in the cache
	got, err = group.Get(ctx, "beta")
	if err != nil {
		t.Fatal(err)
	}
	if string(expect) != string(got) {
		t.Errorf("expect to get %s, but got %s", string(expect), string(got))
	}
	if load != 3 {
		t.Errorf("Expect to load data once, but got %d", load)
	}

	// Ensure the first has been evicted
	got, err = group.Get(ctx, "alpha")
	if err != nil {
		t.Fatal(err)
	}
	if string(expect) != string(got) {
		t.Errorf("expect to get %s, but got %s", string(expect), string(got))
	}
	if load != 4 {
		t.Errorf("Expect to load data once, but got %d", load)
	}
}

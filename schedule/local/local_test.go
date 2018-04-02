package local_test

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stairlin/lego/schedule/local"
)

func Test_Init(t *testing.T) {
	c := local.Config{
		DB: strings.ToLower(t.Name()) + ".db",
	}
	scheduler := local.NewScheduler(c)
	defer os.Remove(c.DB)

	if err := scheduler.Start(); err != nil {
		t.Fatal("cannot start scheduler", err)
	}

	if err := scheduler.Close(); err != nil {
		t.Fatal("cannot stop scheduler", err)
	}
}

func Test_At(t *testing.T) {
	c := local.Config{
		DB: strings.ToLower(t.Name()) + ".db",
	}
	scheduler := local.NewScheduler(c)
	defer os.Remove(c.DB)

	if err := scheduler.Start(); err != nil {
		t.Fatal("cannot start scheduler", err)
	}

	id, err := scheduler.In(time.Second, "foo", nil)
	if err != nil {
		t.Error("cannot add job", err)
	}
	if id == "" {
		t.Error("expect id to be present")
	}

	if err := scheduler.Close(); err != nil {
		t.Fatal("cannot stop scheduler", err)
	}
}

func Test_In(t *testing.T) {
	c := local.Config{
		DB: strings.ToLower(t.Name()) + ".db",
	}
	scheduler := local.NewScheduler(c)
	defer os.Remove(c.DB)

	if err := scheduler.Start(); err != nil {
		t.Fatal("cannot start scheduler", err)
	}

	id, err := scheduler.At(time.Now().Add(time.Second), "foo", nil)
	if err != nil {
		t.Error("cannot add job", err)
	}
	if id == "" {
		t.Error("expect id to be present")
	}

	if err := scheduler.Close(); err != nil {
		t.Fatal("cannot stop scheduler", err)
	}
}

func Test_Register(t *testing.T) {
	c := local.Config{
		DB: strings.ToLower(t.Name()) + ".db",
	}
	scheduler := local.NewScheduler(c)
	defer os.Remove(c.DB)

	if err := scheduler.Start(); err != nil {
		t.Fatal("cannot start scheduler", err)
	}

	dereg, err := scheduler.Register("foo", func(id string, data []byte) error {
		t.Error("unexpected callback")
		return nil
	})
	if err != nil {
		t.Fatal("cannot register callback")
	}

	// Attempt to register a duplicate
	_, err = scheduler.Register("foo", func(id string, data []byte) error {
		t.Error("unexpected callback")
		return nil
	})
	if err == nil {
		t.Error("expect duplicate registration to return an error")
	}

	// De-register
	dereg()

	// Attempt to register Again
	_, err = scheduler.Register("foo", func(id string, data []byte) error {
		t.Error("unexpected callback")
		return nil
	})
	if err != nil {
		t.Fatal("cannot re-register callback")
	}

	if err := scheduler.Close(); err != nil {
		t.Fatal("cannot stop scheduler", err)
	}
}

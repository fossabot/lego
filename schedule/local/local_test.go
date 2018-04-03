package local_test

import (
	"context"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stairlin/lego/schedule/local"
)

func Test_Init(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

	c := local.Config{
		DB: strings.ToLower(t.Name()) + ".db",
	}
	scheduler := local.NewScheduler(c)
	defer os.Remove(c.DB)

	if err := scheduler.Start(); err != nil {
		t.Fatal("cannot start scheduler", err)
	}

	id, err := scheduler.In(context.TODO(), time.Second, "foo", nil)
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
	t.Parallel()

	c := local.Config{
		DB: strings.ToLower(t.Name()) + ".db",
	}
	scheduler := local.NewScheduler(c)
	defer os.Remove(c.DB)

	if err := scheduler.Start(); err != nil {
		t.Fatal("cannot start scheduler", err)
	}

	id, err := scheduler.At(context.TODO(), time.Now().Add(time.Second), "foo", nil)
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
	t.Parallel()

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

// Test_DequeueValidJobs ensures that only scheduled now or in the past
// are being executed
func Test_DequeueValidJobs(t *testing.T) {
	t.Parallel()

	c := local.Config{
		DB: strings.ToLower(t.Name()) + ".db",
	}
	scheduler := local.NewScheduler(c)
	defer os.Remove(c.DB)

	if err := scheduler.Start(); err != nil {
		t.Fatal("cannot start scheduler", err)
	}

	expect := []byte("data dawg")
	var callbackCount uint32

	dereg, err := scheduler.Register("foo", func(id string, data []byte) error {
		atomic.AddUint32(&callbackCount, 1)
		if id == "" {
			t.Error("expect id to not be empty")
		}
		if string(data) != string(expect) {
			t.Errorf("expect data %s, but got %s", expect, data)
		}
		return nil
	})
	if err != nil {
		t.Fatal("cannot register callback")
	}
	defer dereg()

	if _, err := scheduler.In(context.TODO(), time.Millisecond*150, "foo", expect); err != nil {
		t.Fatal("cannot schedule job")
	}
	if _, err := scheduler.In(context.TODO(), time.Millisecond*300, "foo", expect); err != nil {
		t.Fatal("cannot schedule job")
	}

	if _, err := scheduler.In(context.TODO(), time.Second*5, "foo", expect); err != nil {
		t.Fatal("cannot schedule job")
	}
	if _, err := scheduler.In(context.TODO(), time.Second*10, "foo", expect); err != nil {
		t.Fatal("cannot schedule job")
	}
	if _, err := scheduler.In(context.TODO(), time.Second*15, "foo", expect); err != nil {
		t.Fatal("cannot schedule job")
	}
	if _, err := scheduler.In(context.TODO(), time.Second*20, "foo", expect); err != nil {
		t.Fatal("cannot schedule job")
	}

	time.Sleep(time.Millisecond * 10)

	if _, err := scheduler.In(context.TODO(), time.Millisecond*100, "foo", expect); err != nil {
		t.Fatal("cannot schedule job")
	}

	time.Sleep(time.Millisecond * 500)

	var expectCalls uint32 = 3
	if atomic.LoadUint32(&callbackCount) != expectCalls {
		t.Errorf("expect fn to be called back %d times, but got %d",
			expectCalls, callbackCount,
		)
	}
}

// Test_LeaveFutureJobs ensures that future jobs are not executed
func Test_LeaveFutureJobs(t *testing.T) {
	t.Parallel()

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
	defer dereg()

	if _, err := scheduler.In(context.TODO(), time.Second*30, "foo", nil); err != nil {
		t.Fatal("cannot schedule job")
	}
	if _, err := scheduler.In(context.TODO(), time.Second*60, "foo", nil); err != nil {
		t.Fatal("cannot schedule job")
	}
	if _, err := scheduler.In(context.TODO(), time.Second*120, "foo", nil); err != nil {
		t.Fatal("cannot schedule job")
	}

	time.Sleep(time.Millisecond * 10)
}

func Test_LoadLog(t *testing.T) {
	t.Parallel()

	c := local.Config{
		DB:            strings.ToLower(t.Name()) + ".db",
		InitialWindow: time.Hour,
	}
	scheduler := local.NewScheduler(c)
	defer os.Remove(c.DB)

	// Create first log
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
	defer dereg()
	time.Sleep(500 * time.Millisecond)
	if err := scheduler.Close(); err != nil {
		t.Fatal("cannot stop scheduler", err)
	}

	// Test if it has been applied
	scheduler = local.NewScheduler(c)
	if err := scheduler.Start(); err != nil {
		t.Fatal("cannot start scheduler", err)
	}
	if _, err := scheduler.At(context.TODO(), time.Now().Add(-1*time.Minute), "foo", nil); err != nil {
		t.Fatal("cannot schedule job")
	}
	time.Sleep(500 * time.Millisecond)
}

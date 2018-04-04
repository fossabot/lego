package inmem_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stairlin/lego/schedule/inmem"
)

func TestInMem_Init(t *testing.T) {
	scheduler := inmem.NewScheduler()
	if err := scheduler.Start(); err != nil {
		t.Fatal("cannot start scheduler", err)
	}

	if err := scheduler.Close(); err != nil {
		t.Fatal("cannot stop scheduler", err)
	}
}

func TestInMem_ScheduleJob(t *testing.T) {
	scheduler := inmem.NewScheduler()
	if err := scheduler.Start(); err != nil {
		t.Fatal("cannot start scheduler", err)
	}

	for i := 0; i < 20; i++ {
		id, err := scheduler.In(context.TODO(), time.Millisecond, "foo", nil)
		if err != nil {
			t.Fatal("cannot schedule new job", err)
		}
		if id == "" {
			t.Errorf("expect job ID to not be empty")
		}
	}

	time.Sleep(time.Millisecond * 100)

	if err := scheduler.Close(); err != nil {
		t.Fatal("cannot stop scheduler", err)
	}
}

func TestInMem_HandleFunc(t *testing.T) {
	scheduler := inmem.NewScheduler()
	if err := scheduler.Start(); err != nil {
		t.Fatal("cannot start scheduler", err)
	}

	dereg, err := scheduler.HandleFunc("foo", func(id string, data []byte) error {
		t.Error("unexpected callback")
		return nil
	})
	if err != nil {
		t.Fatal("cannot register callback")
	}

	// Attempt to register a duplicate
	_, err = scheduler.HandleFunc("foo", func(id string, data []byte) error {
		t.Error("unexpected callback")
		return nil
	})
	if err == nil {
		t.Error("expect duplicate registration to return an error")
	}

	// De-register
	dereg()

	// Attempt to register Again
	_, err = scheduler.HandleFunc("foo", func(id string, data []byte) error {
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

// TestInMem_DequeueValidJobs ensures that only scheduled now or in the past
// are being executed
func TestInMem_DequeueValidJobs(t *testing.T) {
	scheduler := inmem.NewScheduler()
	if err := scheduler.Start(); err != nil {
		t.Fatal("cannot start scheduler", err)
	}

	expect := []byte("data dawg")
	var callbackCount uint32

	dereg, err := scheduler.HandleFunc("foo", func(id string, data []byte) error {
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

	if _, err := scheduler.At(context.TODO(), time.Now(), "foo", expect); err != nil {
		t.Fatal("cannot schedule job")
	}
	if _, err := scheduler.At(context.TODO(), time.Now(), "foo", expect); err != nil {
		t.Fatal("cannot schedule job")
	}
	if _, err := scheduler.In(context.TODO(), time.Second, "foo", expect); err != nil {
		t.Fatal("cannot schedule job")
	}
	if _, err := scheduler.In(context.TODO(), time.Second*2, "foo", expect); err != nil {
		t.Fatal("cannot schedule job")
	}
	if _, err := scheduler.In(context.TODO(), time.Second*4, "foo", expect); err != nil {
		t.Fatal("cannot schedule job")
	}
	if _, err := scheduler.In(context.TODO(), time.Second*8, "foo", expect); err != nil {
		t.Fatal("cannot schedule job")
	}

	time.Sleep(time.Millisecond * 10)

	if _, err := scheduler.At(context.TODO(), time.Now(), "foo", expect); err != nil {
		t.Fatal("cannot schedule job")
	}

	time.Sleep(time.Millisecond * 150)

	var expectCalls uint32 = 3
	if atomic.LoadUint32(&callbackCount) != expectCalls {
		t.Errorf("expect fn to be called back %d times, but got %d",
			expectCalls, callbackCount,
		)
	}
}

// TestInMem_LeaveFutureJobs ensures that future jobs are not executed
func TestInMem_LeaveFutureJobs(t *testing.T) {
	scheduler := inmem.NewScheduler()
	if err := scheduler.Start(); err != nil {
		t.Fatal("cannot start scheduler", err)
	}

	dereg, err := scheduler.HandleFunc("foo", func(id string, data []byte) error {
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

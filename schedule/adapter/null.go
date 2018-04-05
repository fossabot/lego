package adapter

import (
	"context"
	"time"

	"github.com/stairlin/lego/ctx/app"
	"github.com/stairlin/lego/schedule"
)

// nullScheduler is a scheduler which does not do anything.
type nullScheduler struct{}

func newNullScheduler() schedule.Scheduler {
	return &nullScheduler{}
}

func (s *nullScheduler) Start(ctx app.Ctx) error {
	return nil
}

func (s *nullScheduler) HandleFunc(
	target string, fn schedule.Fn,
) (deregister func(), err error) {
	return func() {}, nil
}

func (s *nullScheduler) At(
	ctx context.Context,
	t time.Time,
	target string,
	data []byte,
	o ...schedule.JobOption,
) (string, error) {
	return "", nil
}

func (s *nullScheduler) In(
	ctx context.Context,
	d time.Duration,
	target string,
	data []byte,
	o ...schedule.JobOption,
) (string, error) {
	return "", nil
}

func (s *nullScheduler) Drain() {}

func (s *nullScheduler) Close() error {
	return nil
}

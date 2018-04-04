package local

import "sync"

// pool is a pool of goroutines that execute functions
type pool struct {
	wg      sync.WaitGroup
	bucketc chan func()
}

// newPool creates a new pool with n goroutines
func newPool(n int) *pool {
	p := &pool{bucketc: make(chan func())}
	p.wg.Add(n)
	for i := 0; i < n; i++ {
		go p.run()
	}
	return p
}

// Exec executes fn in background
func (p *pool) Exec(fn func()) {
	p.bucketc <- fn
}

// Close drains the bucket and stop
func (p *pool) Close() error {
	p.bucketc <- nil
	close(p.bucketc)
	p.wg.Wait()
	return nil
}

func (p *pool) run() {
	for {
		select {
		case fn := <-p.bucketc:
			if fn == nil {
				p.wg.Done()
				return
			}
			fn()
		}
	}
}

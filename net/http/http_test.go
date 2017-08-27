package http_test

import "sync"

// portSequence returns a sequence of port numbers. It should be used
// for test handlers in order to avoid port clashes
var portSequence = &sequencer{n: 9900}

type sequencer struct {
	mu sync.Mutex
	n  int
}

func (s *sequencer) next() int {
	s.mu.Lock()
	p := s.n
	s.n++
	s.mu.Unlock()
	return p
}

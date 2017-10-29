package testing

import "sync"

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

// portSequence returns a sequence of port numbers. It should be used
// for test handlers in order to avoid port clashes
var portSequence = &sequencer{n: 9800}

// NextPort returns the next supposedly available port number
func NextPort() int {
	return portSequence.next()
}

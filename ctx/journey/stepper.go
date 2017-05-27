package journey

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

const stepperSeparator = "_"

// Stepper is the atomic counter for context log lines
type Stepper struct {
	mu sync.Mutex

	Steps []uint32
	I     int
}

// NewStepper builds a new main stepper
func NewStepper() *Stepper {
	return &Stepper{
		Steps: []uint32{0},
		I:     0,
	}
}

// parseSteps parses a string representation of a stepper
//
// e.g. 100_23_40
func parseSteps(s string) (*Stepper, error) {
	values := strings.Split(s, stepperSeparator)
	steps := make([]uint32, len(values))
	for i, v := range values {
		step, err := strconv.ParseUint(v, 10, 32)
		if err != nil {
			return nil, err
		}
		steps[i] = uint32(step)
	}

	return &Stepper{
		Steps: steps,
		I:     len(steps) - 1,
	}, nil
}

// BranchOff returns a new "child" stepper
func (s *Stepper) BranchOff() *Stepper {
	s.mu.Lock()
	defer s.mu.Unlock()

	atomic.AddUint32(&s.Steps[s.I], 1)

	return &Stepper{
		Steps: append(s.Steps, 0),
		I:     s.I + 1,
	}
}

// Inc increments the current counter
func (s *Stepper) Inc() uint {
	s.mu.Lock()
	defer s.mu.Unlock()

	atomic.AddUint32(&s.Steps[s.I], 1)

	return uint(s.Steps[s.I])
}

// String returns a string representation of the current state
func (s *Stepper) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	var buf bytes.Buffer

	for i, step := range s.Steps {
		buf.WriteString(fmt.Sprintf("%04d", step))

		// Add separator
		if i < s.I {
			buf.WriteString(stepperSeparator)
		}
	}

	return buf.String()
}

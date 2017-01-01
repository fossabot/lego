package journey_test

import (
	"testing"

	"github.com/stairlin/lego/ctx/journey"
)

// TestInc tests whether the last counter is properly incremented
func TestInc(t *testing.T) {
	s := journey.NewStepper()

	res := s.Inc()
	expect := uint(1)
	if res != expect {
		t.Errorf("expect step to be equal %d, but got %d", expect, res)
	}

	res = s.Inc()
	expect = uint(2)
	if res != expect {
		t.Errorf("expect step to be equal %d, but got %d", expect, res)
	}

	s = s.BranchOff()

	res = s.Inc()
	expect = uint(1)
	if res != expect {
		t.Errorf("expect step to be equal %d, but got %d", expect, res)
	}
}

// TestStringTestInc tests the string representation of a stepper
func TestString(t *testing.T) {
	tests := []struct {
		in     *journey.Stepper
		expect string
	}{
		{
			in:     journey.NewStepper(),
			expect: "0000",
		},
		{
			in: &journey.Stepper{
				Steps: []uint32{20},
				I:     0,
			},
			expect: "0020",
		},
		{
			in: &journey.Stepper{
				Steps: []uint32{10, 100, 1000},
				I:     2,
			},
			expect: "0010_0100_1000",
		},
	}

	for i, test := range tests {
		got := test.in.String()
		if got != test.expect {
			t.Errorf("%d - expect String to be equal %s, but got %s", i, test.expect, got)
		}
	}

}

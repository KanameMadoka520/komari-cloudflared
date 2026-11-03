package trafficcounter

import (
	"encoding/json"
	"math"
	"testing"
)

func TestCounterDiscontinuities(t *testing.T) {
	const g int64 = 1 << 30
	cases := []struct {
		name      string
		values    []int64
		want      []int64
		uncertain []bool
	}{
		{"monotonic", []int64{100, 110, 110, 135}, []int64{0, 10, 0, 25}, []bool{false, false, false, false}},
		{"mac dip", []int64{46 * g, 44 * g, 44*g + 100, 46*g + 500}, []int64{0, 0, 0, 500}, []bool{false, true, true, true}},
		{"large jitter", []int64{100 * g, 99 * g, 100*g + 1000}, []int64{0, 0, 1000}, []bool{false, true, true}},
		{"zero is ambiguous", []int64{100, 0, 100, 101}, []int64{0, 0, 0, 1}, []bool{false, true, true, false}},
		{"no throughput cap", []int64{1, 500 * g}, []int64{0, 500*g - 1}, []bool{false, false}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var s State
			for i, v := range tc.values {
				n, d, e := s.Advance(v, v, 0, 0)
				if e != nil || d.Up != tc.want[i] || d.Down != tc.want[i] || d.Uncertain != tc.uncertain[i] {
					t.Fatalf("step %d: %+v %v", i, d, e)
				}
				// Persist/reload on every observation, including pending dips.
				b, _ := json.Marshal(n)
				s = State{}
				if e = json.Unmarshal(b, &s); e != nil {
					t.Fatal(e)
				}
			}
		})
	}
}

func TestExplicitEpochResetAndReplay(t *testing.T) {
	s, _, _ := (State{}).Advance(100, 200, 1, 0)
	s, d, e := s.Advance(5, 7, 2, 0)
	if e != nil || d.Up != 5 || d.Down != 7 || d.Uncertain {
		t.Fatalf("reset: %+v %v", d, e)
	}
	s, d, e = s.Advance(110, 210, 1, 0)
	if e != nil || d.Up != 0 || d.Down != 0 || !d.Uncertain || s.Epoch != 2 {
		t.Fatalf("replayed epoch: %+v %+v %v", s, d, e)
	}
	_, d, e = s.Advance(8, 11, 2, 0)
	if e != nil || d.Up != 3 || d.Down != 4 {
		t.Fatalf("after replay: %+v %v", d, e)
	}
}

func TestCounterMetadataEnrollmentAndLoss(t *testing.T) {
	s, _, _ := (State{}).Advance(100, 200, 0, 0)
	s, d, e := s.Advance(120, 220, 1, 0)
	if e != nil || d.Up != 0 || !d.Uncertain {
		t.Fatalf("enrollment: %+v %v", d, e)
	}
	s, d, e = s.Advance(130, 230, 1, 0)
	if e != nil || d.Up != 10 || d.Uncertain {
		t.Fatalf("enrolled: %+v %v", d, e)
	}
	_, d, e = s.Advance(140, 240, 0, 0)
	if e != nil || d.Up != 0 || !d.Uncertain {
		t.Fatalf("metadata loss: %+v %v", d, e)
	}
}

func TestKnown32BitWrapIncludesTail(t *testing.T) {
	s, _, _ := (State{}).Advance((1<<32)-10, (1<<32)-20, 1, 32)
	_, d, e := s.Advance(5, 7, 1, 32)
	if e != nil || d.Up != 15 || d.Down != 27 || d.Uncertain {
		t.Fatalf("wrap: %+v %v", d, e)
	}
}

func TestCounterDirectionsAreIndependent(t *testing.T) {
	s, _, _ := (State{}).Advance(100, 200, 0, 0)
	_, d, e := s.Advance(50, 230, 0, 0)
	if e != nil || d.Up != 0 || d.Down != 30 || !d.Uncertain {
		t.Fatalf("directional delta: %+v %v", d, e)
	}
}

func TestInvalidCounterMetadata(t *testing.T) {
	for _, tc := range []struct {
		up, down int64
		epoch    uint64
		bits     uint8
	}{
		{-1, 1, 0, 0}, {1, -1, 0, 0}, {1, 1, 0, 32}, {1 << 32, 1, 1, 32}, {1, 1, 1, 16},
	} {
		if _, _, e := (State{}).Advance(tc.up, tc.down, tc.epoch, tc.bits); e == nil {
			t.Fatalf("accepted %+v", tc)
		}
	}
}

// For legacy counter snapshots the confirmed lower bound is exactly the
// increase of the prefix maximum, regardless of dips, repetitions or restarts.
func FuzzLegacyCounterNeverRecountsHistory(f *testing.F) {
	f.Add([]byte{100, 90, 101, 0, 40, 102})
	f.Add([]byte{255, 254, 0, 255})
	f.Fuzz(func(t *testing.T, trace []byte) {
		if len(trace) == 0 {
			return
		}
		var s State
		var sum, first, max int64
		for i, b := range trace {
			v := int64(b) * (math.MaxInt64 / 256)
			if i == 0 {
				first = v
			}
			if v > max {
				max = v
			}
			n, d, e := s.Advance(v, v, 0, 0)
			if e != nil {
				t.Fatal(e)
			}
			s = n
			sum += d.Up
			if sum != max-first {
				t.Fatalf("recounted trace at %d: %d != %d", i, sum, max-first)
			}
		}
	})
}

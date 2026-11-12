package trafficcounter

import (
	"encoding/json"
	"testing"
	"time"
)

func TestObservedRebootResumesWithoutCountingNewSnapshot(t *testing.T) {
	at := time.Unix(100000, 0)
	s, _, err := (State{}).AdvanceObserved(1000000, 2000000, 0, 0, 10000, at)
	if err != nil {
		t.Fatal(err)
	}
	encoded, _ := json.Marshal(s)
	s = State{}
	if err = json.Unmarshal(encoded, &s); err != nil {
		t.Fatal(err)
	}
	s, delta, err := s.AdvanceObserved(100, 200, 0, 0, 30, at.Add(time.Minute))
	if err != nil || delta.Up != 0 || delta.Down != 0 || !delta.Uncertain || s.Up.Pending || s.Down.Pending {
		t.Fatalf("reboot baseline: %+v %+v %v", s, delta, err)
	}
	s, delta, err = s.AdvanceObserved(150, 280, 0, 0, 35, at.Add(65*time.Second))
	if err != nil || delta.Up != 50 || delta.Down != 80 || delta.Uncertain {
		t.Fatalf("post-reboot traffic stalled: %+v %+v %v", s, delta, err)
	}
}

func TestUptimeEvidenceDoesNotRecountInterfaceDipsOrPersistentCounters(t *testing.T) {
	at := time.Unix(100000, 0)
	for _, tc := range []struct {
		name                               string
		nextAt                             time.Duration
		uptime, up, down, wantUp, wantDown int64
		pending, uncertain                 bool
	}{
		{"same boot interface dip", time.Second, 10001, 90, 190, 0, 0, true, true},
		{"reboot in same timestamp second", time.Millisecond, 30, 5, 7, 0, 0, false, true},
		{"uptime unavailable", time.Minute, 0, 90, 190, 0, 0, true, true},
		{"small uptime jitter", time.Minute, 9999, 90, 190, 0, 0, true, true},
		{"persistent software counters after boot", time.Minute, 30, 150, 280, 50, 80, false, false},
		{"only one direction resets", time.Minute, 30, 5, 280, 0, 80, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, _, _ := (State{}).AdvanceObserved(100, 200, 0, 0, 10000, at)
			s, d, e := s.AdvanceObserved(tc.up, tc.down, 0, 0, tc.uptime, at.Add(tc.nextAt))
			if e != nil || d.Up != tc.wantUp || d.Down != tc.wantDown || s.Up.Pending != tc.pending || d.Uncertain != tc.uncertain {
				t.Fatalf("%+v %+v %v", s, d, e)
			}
		})
	}
	s, _, _ := (State{}).AdvanceObserved(100, 200, 0, 0, 10000, at)
	s, _, _ = s.AdvanceObserved(90, 190, 0, 0, 10001, at.Add(time.Second))
	_, d, _ := s.AdvanceObserved(105, 209, 0, 0, 10002, at.Add(2*time.Second))
	if d.Up != 5 || d.Down != 9 {
		t.Fatalf("dip recovery recounted old bytes: %+v", d)
	}
}

func TestExplicitGenerationCannotBeRebasedByUptime(t *testing.T) {
	at := time.Unix(100000, 0)
	s, _, _ := (State{}).AdvanceObserved(100, 200, 3, 0, 10000, at)
	s, d, e := s.AdvanceObserved(5, 7, 0, 0, 30, at.Add(time.Minute))
	if e != nil || s.Epoch != 3 || s.Up.HighWater != 100 || d.Up != 0 || !d.Uncertain {
		t.Fatalf("%+v %+v %v", s, d, e)
	}
	if _, e = s.RebasePending(5, 7); e == nil {
		t.Fatal("manual recovery bypassed explicit epoch")
	}
}

func TestAdministrativeRecoveryOnlyRebasesBlockedDirection(t *testing.T) {
	s, _, _ := (State{}).Advance(100, 200, 0, 0)
	s, _, _ = s.Advance(5, 220, 0, 0)
	s, e := s.RebasePending(5, 220)
	if e != nil || s.Up.HighWater != 5 || s.Down.HighWater != 220 || s.Up.Pending {
		t.Fatalf("%+v %v", s, e)
	}
	_, d, _ := s.Advance(8, 225, 0, 0)
	if d.Up != 3 || d.Down != 5 || d.Uncertain {
		t.Fatalf("new traffic did not resume: %+v", d)
	}
	if _, e = s.RebasePending(5, 220); e == nil {
		t.Fatal("healthy counter was rebased")
	}
}

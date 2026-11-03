package metricstore

import (
	"context"
	"testing"
	"time"

	v2 "github.com/komari-monitor/komari/protocol/v2"
)

// A nonzero aggregate can decrease when a counted interface disappears.
// The retained counter is historical usage, not newly transferred bytes.
func TestInvestigationMacInterfaceCounterDip(t *testing.T) {
	const gib int64 = 1 << 30
	if got := TrafficCounterDelta(44*gib, 46*gib); got != 0 {
		t.Fatalf("interface counter dip injected %d bytes (%.2f GiB); want no invented historical traffic", got, float64(got)/float64(gib))
	}
}

func TestInvestigationMacDipPersistsInMetricHistory(t *testing.T) {
	const gib int64 = 1 << 30
	s := useReportTestStore(t, nil)
	base := time.Now().UTC().Truncate(time.Minute).Add(5 * time.Second)
	r := v2.Report{UUID: "mac-interface-dip", UpdatedAt: base, Uptime: 10000,
		Network: v2.NetworkReport{TotalUp: 17 * gib, TotalDown: 46 * gib, Up: 10000, Down: 100000}}
	if _, err := WriteReport(context.Background(), r); err != nil {
		t.Fatal(err)
	}
	r.UpdatedAt = base.Add(3 * time.Second)
	r.Uptime += 3
	r.Network.TotalDown = 44 * gib
	if _, err := WriteReport(context.Background(), r); err != nil {
		t.Fatal(err)
	}
	assertMetricValues(t, s, MetricTrafficDown, r.UUID, base.Add(-time.Second), base.Add(time.Minute), []float64{0, 0})
}

func TestInvestigationLargeCounterDipRecoveryDoubleCounts(t *testing.T) {
	const gib int64 = 1 << 30
	const mib int64 = 1 << 20
	s := useReportTestStore(t, nil)
	base := time.Now().UTC().Truncate(time.Minute).Add(5 * time.Second)
	r := v2.Report{UUID: "large-counter-jitter", UpdatedAt: base, Uptime: 10000,
		Network: v2.NetworkReport{TotalUp: 100 * gib, TotalDown: 100 * gib}}
	for i, down := range []int64{100 * gib, 99 * gib, 100*gib + mib} {
		r.UpdatedAt = base.Add(time.Duration(i*3) * time.Second)
		r.Network.TotalDown = down
		if _, err := WriteReport(context.Background(), r); err != nil {
			t.Fatal(err)
		}
	}
	assertMetricValues(t, s, MetricTrafficDown, r.UUID, base.Add(-time.Second), base.Add(time.Minute), []float64{0, 0, float64(mib)})
}

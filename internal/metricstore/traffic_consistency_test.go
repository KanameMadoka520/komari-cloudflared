package metricstore

import (
	"context"
	"testing"
	"time"

	v2 "github.com/komari-monitor/komari/protocol/v2"
)

func TestTrafficHighWaterSurvivesCompactionAndRestart(t *testing.T) {
	const gib int64 = 1 << 30
	ctx := context.Background()
	policy := defaultRollupPolicy()
	s := useReportTestStore(t, &policy)
	base := time.Now().UTC().Truncate(time.Minute).Add(5 * time.Second)
	r := v2.Report{UUID: "restore-dip", UpdatedAt: base, Network: v2.NetworkReport{TotalUp: 46 * gib, TotalDown: 46 * gib}}
	for i, v := range []int64{46 * gib, 44 * gib} {
		r.UpdatedAt = base.Add(time.Duration(i*3) * time.Second)
		r.Network.TotalDown = v
		if _, e := WriteReport(ctx, r); e != nil {
			t.Fatal(e)
		}
	}
	if _, e := s.Compact(ctx, base.Add(40*time.Second)); e != nil {
		t.Fatal(e)
	}
	deleteReportTrafficState(r.UUID)
	r.UpdatedAt = base.Add(45 * time.Second)
	r.Network.TotalDown = 44*gib + 1000
	if _, e := WriteReport(ctx, r); e != nil {
		t.Fatal(e)
	}
	assertMetricValues(t, s, MetricTrafficDown, r.UUID, r.UpdatedAt.Add(-time.Second), r.UpdatedAt.Add(time.Second), []float64{0})
	r.UpdatedAt = base.Add(48 * time.Second)
	r.Network.TotalDown = 46*gib + 1000
	if _, e := WriteReport(ctx, r); e != nil {
		t.Fatal(e)
	}
	assertMetricValues(t, s, MetricTrafficDown, r.UUID, r.UpdatedAt.Add(-time.Second), r.UpdatedAt.Add(time.Second), []float64{1000})
}

func TestMetricHistoryUsesServerAcceptedDeltaAndPreservesZero(t *testing.T) {
	ctx := context.Background()
	s := useReportTestStore(t, nil)
	base := time.Now().UTC().Truncate(time.Minute).Add(5 * time.Second)
	r := v2.Report{UUID: "canonical-delta", UpdatedAt: base, Network: v2.NetworkReport{TotalUp: 100, TotalDown: 200}, Traffic: &v2.TrafficObservation{Up: 5, Down: 0, HighWaterUp: 100, HighWaterDown: 200, Uncertain: true}}
	if _, e := WriteReport(ctx, r); e != nil {
		t.Fatal(e)
	}
	assertMetricValues(t, s, MetricTrafficUp, r.UUID, base.Add(-time.Second), base.Add(time.Second), []float64{5})
	records, e := GetRecordsByClientAndTime(ctx, r.UUID, base.Add(-time.Second), base.Add(time.Minute))
	if e != nil || len(records) != 1 {
		t.Fatalf("records %+v %v", records, e)
	}
	if !records[0].TrafficUpPresent || !records[0].TrafficDownPresent || records[0].TrafficDown != 0 || records[0].TrafficUncertain != 1 {
		t.Fatalf("presence/quality lost: %+v", records[0])
	}
}

func TestCanonicalGapDoesNotAttributeOfflineUsageToOneBucket(t *testing.T) {
	ctx := context.Background()
	s := useReportTestStore(t, nil)
	base := time.Now().UTC().Truncate(time.Minute).Add(5 * time.Second)
	r := v2.Report{UUID: "canonical-gap", UpdatedAt: base, Network: v2.NetworkReport{TotalUp: 100, TotalDown: 200}, Traffic: &v2.TrafficObservation{Up: 80, Down: 90, HighWaterUp: 100, HighWaterDown: 200, Gap: true}}
	if _, e := WriteReport(ctx, r); e != nil {
		t.Fatal(e)
	}
	assertMetricValues(t, s, MetricTrafficUp, r.UUID, base.Add(-time.Second), base.Add(time.Second), []float64{0})
	assertMetricValues(t, s, MetricTrafficUncertain, r.UUID, base.Add(-time.Second), base.Add(time.Second), []float64{1})
}

func TestCanonicalTotalsRecoverLostQueueReportsAfterRestart(t *testing.T) {
	ctx := context.Background()
	s := useReportTestStore(t, nil)
	base := time.Now().UTC().Truncate(time.Minute).Add(5 * time.Second)
	r := v2.Report{UUID: "lost-queue", UpdatedAt: base, Network: v2.NetworkReport{TotalUp: 100, TotalDown: 200}, Traffic: &v2.TrafficObservation{HasLifetime: true, LifetimeUp: 100, LifetimeDown: 200, HighWaterUp: 100, HighWaterDown: 200}}
	if _, err := WriteReport(ctx, r); err != nil {
		t.Fatal(err)
	}
	// The main DB accepted 120/230, but the process died before that queued sample
	// reached metrics. Recovery must include its 20/30 exactly once.
	deleteReportTrafficState(r.UUID)
	r.UpdatedAt = base.Add(6 * time.Second)
	r.Network.TotalUp, r.Network.TotalDown = 125, 237
	r.Traffic = &v2.TrafficObservation{Up: 5, Down: 7, HasLifetime: true, LifetimeUp: 125, LifetimeDown: 237, PreviousAt: base.Add(3 * time.Second), HighWaterUp: 125, HighWaterDown: 237}
	if _, err := WriteReport(ctx, r); err != nil {
		t.Fatal(err)
	}
	assertMetricValues(t, s, MetricTrafficUp, r.UUID, r.UpdatedAt.Add(-time.Second), r.UpdatedAt.Add(time.Second), []float64{25})
	assertMetricValues(t, s, MetricTrafficDown, r.UUID, r.UpdatedAt.Add(-time.Second), r.UpdatedAt.Add(time.Second), []float64{37})
	assertMetricValues(t, s, MetricTrafficUncertain, r.UUID, r.UpdatedAt.Add(-time.Second), r.UpdatedAt.Add(time.Second), []float64{1})
	previous := r.UpdatedAt
	r.UpdatedAt = base.Add(9 * time.Second)
	r.Traffic = &v2.TrafficObservation{Up: 2, Down: 3, HasLifetime: true, LifetimeUp: 127, LifetimeDown: 240, PreviousAt: previous, HighWaterUp: 127, HighWaterDown: 240}
	if _, err := WriteReport(ctx, r); err != nil {
		t.Fatal(err)
	}
	assertMetricValues(t, s, MetricTrafficUp, r.UUID, r.UpdatedAt.Add(-time.Second), r.UpdatedAt.Add(time.Second), []float64{2})
	assertMetricValues(t, s, MetricTrafficUncertain, r.UUID, r.UpdatedAt.Add(-time.Second), r.UpdatedAt.Add(time.Second), []float64{0})
}

func TestRestartDoesNotInventGapFromTimestampPrecision(t *testing.T) {
	ctx := context.Background()
	s := useReportTestStore(t, nil)
	// Main DB observations retain nanoseconds; persisted metric timestamps use
	// milliseconds. Restoring the same flushed sample must not imply a lost one.
	base := time.Now().UTC().Truncate(time.Minute).Add(5*time.Second + 123456789*time.Nanosecond)
	r := v2.Report{UUID: "precise-restart", UpdatedAt: base, Network: v2.NetworkReport{TotalUp: 100, TotalDown: 200}, Traffic: &v2.TrafficObservation{HasLifetime: true, LifetimeUp: 100, LifetimeDown: 200, HighWaterUp: 100, HighWaterDown: 200}}
	if _, err := WriteReport(ctx, r); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Compact(ctx, base.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	deleteReportTrafficState(r.UUID)
	r.UpdatedAt = base.Add(3 * time.Second)
	r.Network.TotalUp, r.Network.TotalDown = 105, 207
	r.Traffic = &v2.TrafficObservation{Up: 5, Down: 7, HasLifetime: true, LifetimeUp: 105, LifetimeDown: 207, HighWaterUp: 105, HighWaterDown: 207, PreviousAt: base}
	if _, err := WriteReport(ctx, r); err != nil {
		t.Fatal(err)
	}
	assertMetricValues(t, s, MetricTrafficUp, r.UUID, r.UpdatedAt.Add(-time.Second), r.UpdatedAt.Add(time.Second), []float64{5})
	assertMetricValues(t, s, MetricTrafficUncertain, r.UUID, r.UpdatedAt.Add(-time.Second), r.UpdatedAt.Add(time.Second), []float64{0})
}

package notifier

import (
	"math"
	"testing"
)

func TestReportHonorsPresentZeroForGrowingCounter(t *testing.T) {
	previous := trafficDeltaRecord{NetTotalUp: 100, NetTotalDown: 200}
	rows := []trafficDeltaRecord{{NetTotalUp: 150, NetTotalDown: 300, TrafficUpPresent: true, TrafficDownPresent: true}}
	u, d := sumTrafficDeltas(rows, &previous)
	if u != 0 || d != 0 {
		t.Fatalf("recreated excluded interval: %d/%d", u, d)
	}
}

func TestReportMissingLegacyDeltasUseHighWater(t *testing.T) {
	previous := trafficDeltaRecord{NetTotalDown: 100}
	rows := []trafficDeltaRecord{{NetTotalDown: 90}, {NetTotalDown: 101}}
	_, d := sumTrafficDeltas(rows, &previous)
	if d != 1 {
		t.Fatalf("legacy recovery counted %d, want 1", d)
	}
}

func TestReportSaturatesRatherThanWrappingUsage(t *testing.T) {
	rows := []trafficDeltaRecord{{TrafficDown: math.MaxInt64, TrafficDownPresent: true}, {TrafficDown: 100, TrafficDownPresent: true}}
	_, d := sumTrafficDeltas(rows, nil)
	if d != math.MaxInt64 {
		t.Fatal(d)
	}
}

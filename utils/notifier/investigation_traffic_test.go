package notifier

import (
	"testing"
	"time"
)

func TestInvestigationReportRecreatesZeroedCounterDip(t *testing.T) {
	const gib int64 = 1 << 30
	at := time.Unix(1000, 0).UTC()
	prev := trafficDeltaRecord{Time: at, NetTotalDown: 46 * gib}
	rows := []trafficDeltaRecord{{Time: at.Add(3 * time.Second), NetTotalDown: 44 * gib, TrafficDown: 0}}
	_, down := sumTrafficDeltas(rows, &prev)
	if down != 0 {
		t.Fatalf("report fallback recreated %.2f GiB from a zero stored delta", float64(down)/float64(gib))
	}
}

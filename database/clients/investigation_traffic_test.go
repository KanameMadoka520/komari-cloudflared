package clients

import (
	"testing"
	"time"

	"github.com/komari-monitor/komari/database/models"
	v2 "github.com/komari-monitor/komari/protocol/v2"
)

func TestInvestigationLifetimeCounterDipInflatesUsage(t *testing.T) {
	const gib int64 = 1 << 30
	at := time.Unix(1000, 0).UTC()
	c := models.Client{TrafficLifetimeAt: &at, TrafficLifetimeRawDown: 46 * gib, TrafficLifetimeDown: 100 * gib}
	r := v2.Report{UpdatedAt: at.Add(3 * time.Second), Uptime: 10003, Network: v2.NetworkReport{TotalDown: 44 * gib, Down: 100000}}
	advanceLifetime(&c, r)
	if c.TrafficLifetimeDown != 100*gib {
		t.Fatalf("lifetime inflated by %.2f GiB", float64(c.TrafficLifetimeDown-100*gib)/float64(gib))
	}
}

func TestInvestigationBillingCounterDipInflatesUsage(t *testing.T) {
	const gib int64 = 1 << 30
	at := time.Unix(1000, 0).UTC()
	c := models.Client{TrafficResetAt: &at, TrafficObservedAt: &at, TrafficResetDown: 46 * gib, TrafficUsedDown: 10 * gib}
	r := v2.Report{UpdatedAt: at.Add(3 * time.Second), Uptime: 10003, Network: v2.NetworkReport{TotalDown: 44 * gib, Down: 100000}}
	advanceTraffic(&c, r)
	if c.TrafficUsedDown != 10*gib {
		t.Fatalf("billing usage inflated by %.2f GiB", float64(c.TrafficUsedDown-10*gib)/float64(gib))
	}
}

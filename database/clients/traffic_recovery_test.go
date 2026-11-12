package clients

import (
	"testing"
	"time"

	"github.com/komari-monitor/komari/database/models"
	v2 "github.com/komari-monitor/komari/protocol/v2"
)

func TestObservedRebootPreservesLifetimeAndBilling(t *testing.T) {
	at := time.Unix(100000, 0).UTC()
	c := models.Client{}
	r := v2.Report{UpdatedAt: at, Uptime: 10000, Network: v2.NetworkReport{TotalUp: 1000000, TotalDown: 2000000}}
	if _, e := advanceClientTraffic(&c, &r); e != nil {
		t.Fatal(e)
	}
	c.TrafficResetAt, c.TrafficObservedAt = &at, &at
	c.TrafficUsedUp, c.TrafficUsedDown = 400, 600
	initial := int64(12345)
	c.TrafficInitialTotal = &initial
	c.TrafficLimitType = "sum"
	r.UpdatedAt, r.Uptime, r.Network.TotalUp, r.Network.TotalDown = at.Add(time.Minute), 30, 100, 200
	if _, e := advanceClientTraffic(&c, &r); e != nil {
		t.Fatal(e)
	}
	if c.TrafficLifetimeUp != 1000000 || c.TrafficLifetimeDown != 2000000 || c.TrafficUsedUp != 400 || c.TrafficUsedDown != 600 || !c.TrafficIncomplete {
		t.Fatalf("reboot changed totals: %+v", c)
	}
	r.UpdatedAt, r.Uptime, r.Network.TotalUp, r.Network.TotalDown = at.Add(65*time.Second), 35, 110, 220
	if _, e := advanceClientTraffic(&c, &r); e != nil {
		t.Fatal(e)
	}
	if c.TrafficLifetimeUp != 1000010 || c.TrafficLifetimeDown != 2000020 || EffectiveTrafficTotal(c, 0, 0) != 13375 || r.Traffic.Uncertain {
		t.Fatalf("new traffic stalled: %+v %+v", c, r.Traffic)
	}
	if pending, _ := TrafficCounterHealth(c); pending {
		t.Fatal("recovered counter still pending")
	}
}

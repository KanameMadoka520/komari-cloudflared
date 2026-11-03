package clients

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/komari-monitor/komari/database/models"
	v2 "github.com/komari-monitor/komari/protocol/v2"
)

func TestOneTrafficDecisionSurvivesReloadAndCalibration(t *testing.T) {
	const gib int64 = 1 << 30
	at := time.Unix(1000, 0).UTC()
	c := models.Client{TrafficLifetimeAt: &at, TrafficLifetimeRawUp: 100, TrafficLifetimeRawDown: 46 * gib, TrafficLifetimeUp: 1000, TrafficLifetimeDown: 100 * gib, TrafficResetAt: &at, TrafficObservedAt: &at, TrafficResetUp: 100, TrafficResetDown: 46 * gib, TrafficUsedUp: 10, TrafficUsedDown: 2 * gib}
	r := v2.Report{UpdatedAt: at.Add(time.Second), Network: v2.NetworkReport{TotalUp: 110, TotalDown: 44 * gib}}
	if ok, e := advanceClientTraffic(&c, &r); e != nil || !ok {
		t.Fatal(ok, e)
	}
	if r.Traffic.Up != 10 || r.Traffic.Down != 0 || !r.Traffic.Uncertain || !c.TrafficIncomplete {
		t.Fatalf("decision %+v %+v", r.Traffic, c)
	}
	if c.TrafficLifetimeDown != 100*gib || c.TrafficUsedDown != 2*gib || c.TrafficUsedUp != 20 {
		t.Fatal("accounting diverged")
	}
	// The serialized state, not the most recent lower raw counter, is restored.
	var persisted map[string]any
	if e := json.Unmarshal([]byte(c.TrafficCounterState), &persisted); e != nil {
		t.Fatal(e)
	}
	initial := int64(123)
	c.TrafficInitialTotal = &initial
	c.TrafficUsedUp, c.TrafficUsedDown = 0, 0
	r.UpdatedAt = r.UpdatedAt.Add(time.Second)
	r.Network.TotalDown = 44*gib + 1000
	if _, e := advanceClientTraffic(&c, &r); e != nil {
		t.Fatal(e)
	}
	if r.Traffic.Down != 0 || EffectiveTrafficTotal(c, 0, 0) != 123 {
		t.Fatal("calibration discarded the counter high-water mark")
	}
	r.UpdatedAt = r.UpdatedAt.Add(time.Second)
	r.Network.TotalDown = 46*gib + 1000
	if _, e := advanceClientTraffic(&c, &r); e != nil {
		t.Fatal(e)
	}
	if r.Traffic.Down != 1000 || c.TrafficUsedDown != 1000 || c.TrafficLifetimeDown != 100*gib+1000 {
		t.Fatal("recovery double counted")
	}
}

func TestTrafficDecisionCannotBeSuppliedByAgentJSON(t *testing.T) {
	var r v2.Report
	if e := json.Unmarshal([]byte(`{"traffic":{"Up":99999},"Traffic":{"Down":99999},"network":{"totalUp":1,"totalDown":2}}`), &r); e != nil {
		t.Fatal(e)
	}
	if r.Traffic != nil {
		t.Fatal("agent supplied server-owned traffic deltas")
	}
}

func TestCorruptTrafficStateDoesNotResetAccounting(t *testing.T) {
	c := models.Client{TrafficCounterState: "{}"}
	r := v2.Report{UpdatedAt: time.Now(), Network: v2.NetworkReport{TotalUp: 100, TotalDown: 200}}
	if _, e := advanceClientTraffic(&c, &r); e == nil {
		t.Fatal("silently reset malformed state")
	}
}

package clients

import (
	"testing"
	"time"

	"github.com/komari-monitor/komari/database/models"
	v2 "github.com/komari-monitor/komari/protocol/v2"
)

func TestEffectiveTrafficUsesConfiguredCycle(t *testing.T) {
	resetAt := time.Unix(100, 0)
	client := models.Client{
		TrafficResetAt:     &resetAt,
		TrafficResetUp:     1000,
		TrafficResetDown:   2000,
		TrafficInitialUp:   3 * 1024,
		TrafficInitialDown: 5 * 1024,
	}
	up, down := EffectiveTraffic(client, 1500, 2300)
	if up != 3*1024+500 || down != 5*1024+300 {
		t.Fatalf("unexpected cycle usage: up=%d down=%d", up, down)
	}
}

func TestEffectiveTrafficHandlesAgentCounterReset(t *testing.T) {
	resetAt := time.Unix(100, 0)
	client := models.Client{TrafficResetAt: &resetAt, TrafficResetUp: 1000, TrafficResetDown: 2000, TrafficInitialUp: 7, TrafficInitialDown: 9}
	up, down := EffectiveTraffic(client, 12, 34)
	if up != 19 || down != 43 {
		t.Fatalf("unexpected reboot-adjusted usage: up=%d down=%d", up, down)
	}
}

func TestEffectiveTrafficKeepsLegacyClientsUnchanged(t *testing.T) {
	up, down := EffectiveTraffic(models.Client{}, 123, 456)
	if up != 123 || down != 456 {
		t.Fatalf("unexpected legacy usage: up=%d down=%d", up, down)
	}
}

func TestCycleSurvivesMultipleCounterResets(t *testing.T) {
	at := time.Unix(100, 0).UTC()
	c := models.Client{UUID: "node", TrafficResetAt: &at, TrafficObservedAt: &at, TrafficResetUp: 1000, TrafficResetDown: 2000, TrafficUsedUp: 40, TrafficUsedDown: 60}
	samples := []struct{ up, down, wantUp, wantDown int64 }{
		{1100, 2200, 140, 260}, {10, 20, 150, 280}, {1200, 2300, 1340, 2560}, {0, 10, 1340, 2570}, {2, 12, 1342, 2572},
	}
	for i, s := range samples {
		r := v2.Report{UpdatedAt: at.Add(time.Duration(i+1) * time.Second), Network: v2.NetworkReport{TotalUp: s.up, TotalDown: s.down}}
		if !advanceTraffic(&c, r) {
			t.Fatal("report skipped")
		}
		if up, down := EffectiveTraffic(c, s.up, s.down); up != s.wantUp || down != s.wantDown {
			t.Fatalf("sample %d: %d/%d want %d/%d", i, up, down, s.wantUp, s.wantDown)
		}
		if advanceTraffic(&c, r) {
			t.Fatal("duplicate was counted")
		}
		r.UpdatedAt = at
		if advanceTraffic(&c, r) {
			t.Fatal("old report was counted")
		}
	}
}

func TestProviderTotalUsesBillingRuleWithoutInventingSplit(t *testing.T) {
	at := time.Unix(100, 0).UTC()
	initial := int64(120 * 1024 * 1024 * 1024)
	c := models.Client{TrafficResetAt: &at, TrafficObservedAt: &at, TrafficInitialTotal: &initial,
		TrafficResetUp: 1000, TrafficResetDown: 2000}
	report := v2.Report{UpdatedAt: at.Add(time.Second), Network: v2.NetworkReport{TotalUp: 1000 + 1024*1024*1024, TotalDown: 2000 + 2*1024*1024*1024}}
	advanceTraffic(&c, report)
	for _, tc := range []struct {
		rule  string
		delta int64
	}{{"sum", 3}, {"up", 1}, {"down", 2}, {"max", 2}, {"min", 1}, {"SUM", 3}, {"", 2}} {
		c.TrafficLimitType = tc.rule
		if got := EffectiveTrafficTotal(c, 0, 0); got != initial+tc.delta*1024*1024*1024 {
			t.Fatalf("rule %q: got %d", tc.rule, got)
		}
	}
	if up, down := EffectiveTraffic(c, 0, 0); up != 1024*1024*1024 || down != 2*1024*1024*1024 {
		t.Fatal("provider baseline was fabricated as directional traffic")
	}
	c.TrafficLimitType = "sum"
	report.UpdatedAt = report.UpdatedAt.Add(time.Second)
	report.Network.TotalUp, report.Network.TotalDown = 5, 7
	advanceTraffic(&c, report)
	if got := EffectiveTrafficTotal(c, 0, 0); got != initial+3*1024*1024*1024+12 {
		t.Fatalf("counter reset lost provider usage: %d", got)
	}
	initial = 0
	if got := EffectiveTrafficTotal(c, 0, 0); got != 3*1024*1024*1024+12 {
		t.Fatalf("zero baseline failed: %d", got)
	}
	initial = 9223372036854775807
	if got := EffectiveTrafficTotal(c, 0, 0); got != initial {
		t.Fatalf("total overflowed: %d", got)
	}
}

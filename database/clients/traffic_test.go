package clients

import (
	"testing"
	"time"

	"github.com/komari-monitor/komari/database/models"
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

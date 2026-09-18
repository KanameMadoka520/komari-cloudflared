package clients

import (
	"fmt"
	"github.com/komari-monitor/komari/internal/metricstore"
	v2 "github.com/komari-monitor/komari/protocol/v2"
	"math"
	"sync"
	"time"

	"github.com/komari-monitor/komari/database/dbcore"
	"github.com/komari-monitor/komari/database/models"
)

// TrafficMu serializes counter observation and admin snapshots through publication
// of the corresponding latest report, preventing a reset from using a stale baseline.
var TrafficMu sync.Mutex

// EffectiveTraffic returns durable cycle usage; the agent's raw counters and
// metric history retain their original meaning.
func EffectiveTraffic(client models.Client, rawUp, rawDown int64) (int64, int64) {
	if client.TrafficResetAt == nil {
		return clampNonNegative(rawUp), clampNonNegative(rawDown)
	}
	if client.TrafficObservedAt != nil {
		return client.TrafficUsedUp, client.TrafficUsedDown
	}
	return effectiveCounter(rawUp, client.TrafficResetUp, client.TrafficInitialUp),
		effectiveCounter(rawDown, client.TrafficResetDown, client.TrafficInitialDown)
}

func effectiveCounter(raw, baseline, initial int64) int64 {
	raw = clampNonNegative(raw)
	baseline = clampNonNegative(baseline)
	initial = clampNonNegative(initial)
	if raw >= baseline {
		return saturatingAdd(initial, raw-baseline)
	}
	return saturatingAdd(initial, raw)
}

func clampNonNegative(value int64) int64 {
	if value < 0 {
		return 0
	}
	return value
}

func saturatingAdd(left, right int64) int64 {
	if right > math.MaxInt64-left {
		return math.MaxInt64
	}
	return left + right
}

// SetTrafficBaseline starts a new usage cycle for one client. raw counters
// are captured at the time of the operation; initial values are what the
// administrator wants the current cycle to display immediately.
func SetTrafficBaseline(uuid string, rawUp, rawDown, initialUp, initialDown int64, at time.Time) error {
	if uuid == "" {
		return fmt.Errorf("invalid client UUID")
	}
	for name, value := range map[string]int64{
		"raw upload counter": rawUp, "raw download counter": rawDown,
		"initial upload usage": initialUp, "initial download usage": initialDown,
	} {
		if value < 0 {
			return fmt.Errorf("%s must be non-negative", name)
		}
	}
	at = at.UTC()
	db := dbcore.GetDBInstance()
	result := db.Model(&models.Client{}).Where("uuid = ?", uuid).Updates(map[string]interface{}{
		"traffic_reset_at":     at,
		"traffic_reset_up":     rawUp,
		"traffic_reset_down":   rawDown,
		"traffic_initial_up":   initialUp,
		"traffic_initial_down": initialDown,
		"updated_at":           at,
		"traffic_used_up":      initialUp,
		"traffic_used_down":    initialDown,
		"traffic_observed_at":  at,
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("client not found: %s", uuid)
	}
	return nil
}

// ObserveTraffic persists only configured cycles. Caller holds TrafficMu.
// Independent upload/download baselines handle Agent resets without losing the
// usage accumulated before the reset; repeated or old samples are ignored.
func ObserveTraffic(report v2.Report) error {
	db := dbcore.GetDBInstance()
	var client models.Client
	result := db.Where("uuid = ? AND traffic_reset_at IS NOT NULL", report.UUID).Limit(1).Find(&client)
	if result.Error != nil || result.RowsAffected == 0 {
		return result.Error
	}
	if !advanceTraffic(&client, report) {
		return nil
	}
	return db.Model(&models.Client{}).Where("uuid = ?", report.UUID).Updates(map[string]any{
		"traffic_reset_up": client.TrafficResetUp, "traffic_reset_down": client.TrafficResetDown,
		"traffic_used_up": client.TrafficUsedUp, "traffic_used_down": client.TrafficUsedDown,
		"traffic_observed_at": client.TrafficObservedAt,
	}).Error
}

func advanceTraffic(client *models.Client, report v2.Report) bool {
	if client.TrafficResetAt == nil {
		return false
	}
	if client.TrafficObservedAt != nil && !report.UpdatedAt.After(*client.TrafficObservedAt) {
		return false
	}
	if client.TrafficObservedAt == nil {
		client.TrafficUsedUp, client.TrafficUsedDown = client.TrafficInitialUp, client.TrafficInitialDown
	}
	client.TrafficUsedUp = saturatingAdd(client.TrafficUsedUp, metricstore.TrafficCounterDelta(report.Network.TotalUp, client.TrafficResetUp))
	client.TrafficUsedDown = saturatingAdd(client.TrafficUsedDown, metricstore.TrafficCounterDelta(report.Network.TotalDown, client.TrafficResetDown))
	client.TrafficResetUp, client.TrafficResetDown = report.Network.TotalUp, report.Network.TotalDown
	at := report.UpdatedAt.UTC()
	client.TrafficObservedAt = &at
	return true
}

package clients

import (
	"fmt"
	"math"
	"time"

	"github.com/komari-monitor/komari/database/dbcore"
	"github.com/komari-monitor/komari/database/models"
)

// EffectiveTraffic applies the administrator's current usage-cycle baseline
// to the lifetime counters reported by an agent. A counter that drops below
// its baseline is treated as an agent reboot/counter reset; the current
// counter then starts a new segment without losing the manually entered usage.
func EffectiveTraffic(client models.Client, rawUp, rawDown int64) (int64, int64) {
	if client.TrafficResetAt == nil {
		return clampNonNegative(rawUp), clampNonNegative(rawDown)
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
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("client not found: %s", uuid)
	}
	return nil
}

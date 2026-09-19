package clients

import (
	"context"
	"fmt"
	"github.com/komari-monitor/komari/internal/metricstore"
	v2 "github.com/komari-monitor/komari/protocol/v2"
	"math"
	"strings"
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

// TrafficByType applies the node's billing rule to directional usage.
func TrafficByType(kind string, up, down int64) int64 {
	up, down = clampNonNegative(up), clampNonNegative(down)
	switch strings.ToLower(kind) {
	case "up":
		return up
	case "down":
		return down
	case "sum":
		return saturatingAdd(up, down)
	case "min":
		return min(up, down)
	default:
		return max(up, down)
	}
}

// EffectiveTrafficTotal includes an independent provider baseline, if set.
// In total mode EffectiveTraffic's directions are increments since calibration.
func EffectiveTrafficTotal(client models.Client, rawUp, rawDown int64) int64 {
	up, down := EffectiveTraffic(client, rawUp, rawDown)
	used := TrafficByType(client.TrafficLimitType, up, down)
	if client.TrafficInitialTotal != nil && client.TrafficResetAt != nil {
		used = saturatingAdd(clampNonNegative(*client.TrafficInitialTotal), used)
	}
	return used
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
	return SetTrafficUsageBaseline(uuid, rawUp, rawDown, initialUp, initialDown, nil, at)
}

// SetTrafficUsageBaseline atomically switches between total and split entry.
// A total baseline cannot be mixed with initial directional usage.
func SetTrafficUsageBaseline(uuid string, rawUp, rawDown, initialUp, initialDown int64, total *int64, at time.Time) error {
	if total != nil && (*total < 0 || initialUp != 0 || initialDown != 0) {
		return fmt.Errorf("total usage must be non-negative and cannot be mixed with directional usage")
	}
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
		"traffic_initial_total": total,
		"traffic_reset_at":      at,
		"traffic_reset_up":      rawUp,
		"traffic_reset_down":    rawDown,
		"traffic_initial_up":    initialUp,
		"traffic_initial_down":  initialDown,
		"updated_at":            at,
		"traffic_used_up":       initialUp,
		"traffic_used_down":     initialDown,
		"traffic_observed_at":   at,
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("client not found: %s", uuid)
	}
	return nil
}

// LifetimeTraffic is the server-retained total, independent of billing resets.
func LifetimeTraffic(client models.Client, rawUp, rawDown int64) (int64, int64) {
	if client.TrafficLifetimeAt == nil {
		return clampNonNegative(rawUp), clampNonNegative(rawDown)
	}
	return client.TrafficLifetimeUp, client.TrafficLifetimeDown
}

func advanceLifetime(client *models.Client, report v2.Report) bool {
	if client.TrafficLifetimeAt != nil && !report.UpdatedAt.After(*client.TrafficLifetimeAt) {
		return false
	}
	up, down := clampNonNegative(report.Network.TotalUp), clampNonNegative(report.Network.TotalDown)
	if client.TrafficLifetimeAt == nil {
		client.TrafficLifetimeUp, client.TrafficLifetimeDown = up, down
	} else {
		// Ignore a large counter dip as jitter and retain the high-water mark,
		// so recovery to the previous value cannot be counted a second time.
		add := func(total, previous *int64, current int64) {
			delta := metricstore.TrafficCounterDelta(current, *previous)
			if current < *previous && current > 0 && delta == 0 {
				return
			}
			*total = saturatingAdd(*total, delta)
			*previous = current
		}
		add(&client.TrafficLifetimeUp, &client.TrafficLifetimeRawUp, up)
		add(&client.TrafficLifetimeDown, &client.TrafficLifetimeRawDown, down)
		at := report.UpdatedAt.UTC()
		client.TrafficLifetimeAt = &at
		return true
	}
	client.TrafficLifetimeRawUp, client.TrafficLifetimeRawDown = up, down
	at := report.UpdatedAt.UTC()
	client.TrafficLifetimeAt = &at
	return true
}

func lifetimeUpdates(client models.Client) map[string]any {
	return map[string]any{
		"traffic_lifetime_up": client.TrafficLifetimeUp, "traffic_lifetime_down": client.TrafficLifetimeDown,
		"traffic_lifetime_raw_up": client.TrafficLifetimeRawUp, "traffic_lifetime_raw_down": client.TrafficLifetimeRawDown,
		"traffic_lifetime_at": client.TrafficLifetimeAt,
	}
}

// InitializeTrafficLifetime runs before accepting reports. Seed existing nodes
// once from retained raw counters, including offline nodes. Never seed from
// manually entered billing usage or from a retention-limited sum of samples.
func InitializeTrafficLifetime(ctx context.Context) error {
	TrafficMu.Lock()
	defer TrafficMu.Unlock()
	db := dbcore.GetDBInstance()
	var pending []models.Client
	if err := db.Where("traffic_lifetime_at IS NULL").Find(&pending).Error; err != nil {
		return err
	}
	if len(pending) == 0 {
		return nil
	}
	ids := make([]string, 0, len(pending))
	for _, c := range pending {
		ids = append(ids, c.UUID)
	}
	at := time.Now().UTC()
	latest, err := metricstore.GetLatestTrafficBefore(ctx, ids, at)
	if err != nil {
		return err
	}
	for _, c := range pending {
		record, ok := latest[c.UUID]
		// Configured cycles persist a raw counter on every observation; prefer
		// this over the metric store's batched sample on the initial upgrade.
		if c.TrafficObservedAt != nil {
			record.NetTotalUp, record.NetTotalDown = c.TrafficResetUp, c.TrafficResetDown
			ok = true
		}
		if !ok {
			continue
		}
		advanceLifetime(&c, v2.Report{UpdatedAt: at, Network: v2.NetworkReport{TotalUp: record.NetTotalUp, TotalDown: record.NetTotalDown}})
		if err := db.Model(&models.Client{}).Where("uuid = ? AND traffic_lifetime_at IS NULL", c.UUID).Updates(lifetimeUpdates(c)).Error; err != nil {
			return err
		}
	}
	return nil
}

// ObserveTraffic persists lifetime and optional billing counters atomically.
// Caller holds TrafficMu. Presence changes never modify either counter set.
func ObserveTraffic(report v2.Report) error {
	db := dbcore.GetDBInstance()
	var client models.Client
	result := db.Where("uuid = ?", report.UUID).Limit(1).Find(&client)
	if result.Error != nil || result.RowsAffected == 0 {
		return result.Error
	}
	updates := map[string]any{}
	if advanceLifetime(&client, report) {
		updates = lifetimeUpdates(client)
	}
	if advanceTraffic(&client, report) {
		updates["traffic_reset_up"], updates["traffic_reset_down"] = client.TrafficResetUp, client.TrafficResetDown
		updates["traffic_used_up"], updates["traffic_used_down"] = client.TrafficUsedUp, client.TrafficUsedDown
		updates["traffic_observed_at"] = client.TrafficObservedAt
	}
	if len(updates) == 0 {
		return nil
	}
	return db.Model(&models.Client{}).Where("uuid = ?", report.UUID).Updates(updates).Error
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

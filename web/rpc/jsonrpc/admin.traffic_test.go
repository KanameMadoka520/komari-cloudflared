package jsonrpc

import (
	"context"
	"encoding/json"
	"github.com/komari-monitor/komari/database/clients"
	"github.com/komari-monitor/komari/database/dbcore"
	"github.com/komari-monitor/komari/database/models"
	"github.com/komari-monitor/komari/pkg/rpc"
	v2 "github.com/komari-monitor/komari/protocol/v2"
	agent "github.com/komari-monitor/komari/web/agent"
	"testing"
	"time"
)

func TestTrafficCalibrationPersistsAndKeepsRawCounters(t *testing.T) {
	db := dbcore.GetDBInstance()
	id := "traffic-test-node"
	if err := db.Create(&models.Client{UUID: id, Token: id, Name: id}).Error; err != nil {
		t.Fatal(err)
	}
	defer db.Delete(&models.Client{}, "uuid = ?", id)
	defer agent.DeleteLatestReport(id)
	report := v2.Report{UUID: id, UpdatedAt: time.Now().UTC(), Network: v2.NetworkReport{TotalUp: 1000, TotalDown: 2000, CounterEpoch: 1}}
	if err := clients.ObserveTraffic(report); err != nil {
		t.Fatal(err)
	}
	agent.RecordReport(report)
	req := &rpc.JsonRpcRequest{Params: map[string]any{"uuid": id, "upload": 100, "download": 200}}
	if _, e := adminSetClientTrafficUsage(context.Background(), req); e != nil {
		t.Fatal(e)
	}
	report.UpdatedAt = time.Now().UTC().Add(time.Second)
	report.Network.TotalUp = 1050
	report.Network.TotalDown = 2075
	if err := clients.ObserveTraffic(report); err != nil {
		t.Fatal(err)
	}
	agent.RecordReport(report)
	c, err := clients.GetClientByUUID(id)
	if err != nil {
		t.Fatal(err)
	}
	if up, down := clients.EffectiveTraffic(c, 0, 0); up != 150 || down != 275 {
		t.Fatalf("durable totals %d/%d", up, down)
	}
	if agent.GetLatestReport()[id].Network.TotalUp != 1050 {
		t.Fatal("raw report changed")
	}
	// RPC latest status carries both independent values.
	result, e := getNodesLatestStatus(rpc.NewContextWithMeta(context.Background(), &rpc.ContextMeta{}), &rpc.JsonRpcRequest{Params: map[string]any{"uuid": id}})
	if e != nil {
		t.Fatal(e)
	}
	b, _ := json.Marshal(result)
	var data map[string]any
	json.Unmarshal(b, &data)
	if data["net_total_up"] != float64(1050) || data["traffic_used_up"] != float64(150) {
		t.Fatalf("status: %s", b)
	}
	// A reboot retains all usage accumulated in this cycle.
	report.UpdatedAt = report.UpdatedAt.Add(time.Second)
	report.Network.TotalUp = 5
	report.Network.TotalDown = 7
	report.Network.CounterEpoch = 2
	if err := clients.ObserveTraffic(report); err != nil {
		t.Fatal(err)
	}
	c, _ = clients.GetClientByUUID(id)
	if up, down := clients.EffectiveTraffic(c, 5, 7); up != 155 || down != 282 {
		t.Fatalf("reboot lost traffic %d/%d", up, down)
	}
	for _, params := range []map[string]any{{"uuid": id}, {"uuid": id, "upload": -1, "download": 0}, {"uuid": id, "upload": 1.5, "download": 0}, {"uuid": id, "upload": nil, "download": 0}} {
		if _, e := adminSetClientTrafficUsage(context.Background(), &rpc.JsonRpcRequest{Params: params}); e == nil {
			t.Fatalf("accepted invalid params %#v", params)
		}
	}
	if err := clients.SaveClient(map[string]any{"uuid": id, "traffic_used_up": 1}); err == nil {
		t.Fatal("generic edit bypassed cycle API")
	}
}

func TestProviderTotalCalibrationAPI(t *testing.T) {
	db := dbcore.GetDBInstance()
	id := "provider-total-test"
	if err := db.Create(&models.Client{UUID: id, Token: id, Name: id, TrafficLimitType: "sum"}).Error; err != nil {
		t.Fatal(err)
	}
	defer db.Delete(&models.Client{}, "uuid = ?", id)
	defer agent.DeleteLatestReport(id)
	report := v2.Report{UUID: id, UpdatedAt: time.Now().UTC(), Network: v2.NetworkReport{TotalUp: 1000, TotalDown: 2000}}
	agent.RecordReport(report)
	call := func(values map[string]any) (any, *rpc.JsonRpcError) {
		values["uuid"] = id
		return adminSetClientTrafficUsage(context.Background(), &rpc.JsonRpcRequest{Params: values})
	}
	result, e := call(map[string]any{"total": 120})
	if e != nil {
		t.Fatal(e)
	}
	snapshot := result.(map[string]any)
	if snapshot["total"] != int64(120) || snapshot["upload"] != int64(0) || snapshot["total_mode"] != true {
		t.Fatalf("bad snapshot: %v", snapshot)
	}
	report.UpdatedAt = time.Now().UTC().Add(time.Second)
	report.Network.TotalUp, report.Network.TotalDown = 1001, 2002
	if err := clients.ObserveTraffic(report); err != nil {
		t.Fatal(err)
	}
	agent.RecordReport(report)
	c, err := clients.GetClientByUUID(id)
	if err != nil {
		t.Fatal(err)
	}
	if c.TrafficInitialTotal == nil || clients.EffectiveTrafficTotal(c, 0, 0) != 123 {
		t.Fatal("total failed to persist")
	}
	ctx := rpc.NewContextWithMeta(context.Background(), &rpc.ContextMeta{})
	result, e = getNodesLatestStatus(ctx, &rpc.JsonRpcRequest{Params: map[string]any{"uuid": id}})
	if e != nil {
		t.Fatal(e)
	}
	b, _ := json.Marshal(result)
	var data map[string]any
	json.Unmarshal(b, &data)
	if data["traffic_used_total"] != float64(123) || data["traffic_total_mode"] != true || data["net_total_up"] != float64(1001) || data["traffic_used_up"] != float64(1) {
		t.Fatalf("status: %s", b)
	}
	for _, bad := range []map[string]any{
		{"total": -1}, {"total": 1.5}, {"total": nil}, {"total": "1 GB"}, {"total": 9007199254740992},
		{"total": 10, "upload": 0}, {"total": 10, "download": nil}, {"total": nil, "upload": 0, "download": 0},
	} {
		if _, e := call(bad); e == nil {
			t.Fatalf("accepted invalid params: %v", bad)
		}
	}
	c, _ = clients.GetClientByUUID(id)
	if clients.EffectiveTrafficTotal(c, 0, 0) != 123 {
		t.Fatal("invalid request changed baseline")
	}
	if err := clients.SaveClient(map[string]any{"uuid": id, "traffic_initial_total": 1}); err == nil {
		t.Fatal("generic edit bypassed total baseline protection")
	}
	if _, e := call(map[string]any{"total": 0}); e != nil {
		t.Fatal(e)
	}
	c, _ = clients.GetClientByUUID(id)
	if c.TrafficInitialTotal == nil || clients.EffectiveTrafficTotal(c, 0, 0) != 0 {
		t.Fatal("zero total did not persist")
	}
	if _, e := call(map[string]any{"upload": 20, "download": 30}); e != nil {
		t.Fatal(e)
	}
	c, _ = clients.GetClientByUUID(id)
	if c.TrafficInitialTotal != nil || clients.EffectiveTrafficTotal(c, 0, 0) != 50 {
		t.Fatal("split did not clear total mode")
	}
	if _, e := call(map[string]any{"total": 120}); e != nil {
		t.Fatal(e)
	}
	if _, e := adminResetClientTraffic(context.Background(), &rpc.JsonRpcRequest{Params: map[string]any{"uuid": id}}); e != nil {
		t.Fatal(e)
	}
	c, _ = clients.GetClientByUUID(id)
	if c.TrafficInitialTotal != nil || clients.EffectiveTrafficTotal(c, 0, 0) != 0 {
		t.Fatal("reset retained total baseline")
	}
}

func TestTrafficRecoveryKeepsTotalsAndRejectsStaleSnapshots(t *testing.T) {
	db := dbcore.GetDBInstance()
	id := "traffic-recovery-test"
	if err := db.Create(&models.Client{UUID: id, Token: id, Name: id, TrafficLimitType: "sum"}).Error; err != nil {
		t.Fatal(err)
	}
	defer db.Delete(&models.Client{}, "uuid = ?", id)
	at := time.Now().UTC().Add(-time.Second)
	report := v2.Report{UUID: id, UpdatedAt: at, Uptime: 10000, Network: v2.NetworkReport{TotalUp: 1000, TotalDown: 2000}}
	if err := clients.ObserveTraffic(report); err != nil {
		t.Fatal(err)
	}
	if err := clients.SetTrafficBaseline(id, 1000, 2000, 123, 456, at); err != nil {
		t.Fatal(err)
	}
	report.UpdatedAt, report.Uptime = time.Now().UTC(), 10001
	report.Network.TotalUp, report.Network.TotalDown = 10, 2020
	if err := clients.ObserveTraffic(report); err != nil {
		t.Fatal(err)
	}
	before, _ := clients.GetClientByUUID(id)
	req := &rpc.JsonRpcRequest{Params: map[string]any{"uuid": id}}
	result, rpcErr := adminRebaseClientTrafficCounters(context.Background(), req)
	if rpcErr != nil {
		t.Fatal(rpcErr)
	}
	if result.(map[string]any)["counter_pending"] != false || result.(map[string]any)["incomplete"] != true {
		t.Fatal(result)
	}
	after, _ := clients.GetClientByUUID(id)
	if before.TrafficLifetimeUp != after.TrafficLifetimeUp || before.TrafficLifetimeDown != after.TrafficLifetimeDown || before.TrafficUsedUp != after.TrafficUsedUp || before.TrafficUsedDown != after.TrafficUsedDown || !before.TrafficResetAt.Equal(*after.TrafficResetAt) {
		t.Fatal("recovery changed billing/lifetime totals or calibration date")
	}
	report.UpdatedAt, report.Uptime = time.Now().UTC().Add(time.Second), 10002
	report.Network.TotalUp, report.Network.TotalDown = 15, 2027
	if err := clients.ObserveTraffic(report); err != nil {
		t.Fatal(err)
	}
	after, _ = clients.GetClientByUUID(id)
	if after.TrafficLifetimeUp != before.TrafficLifetimeUp+5 || after.TrafficUsedUp != before.TrafficUsedUp+5 || after.TrafficUsedDown != before.TrafficUsedDown+7 {
		t.Fatal("post-recovery accounting diverged")
	}
	if _, e := adminRebaseClientTrafficCounters(context.Background(), req); e == nil {
		t.Fatal("healthy counter accepted recovery")
	}
	report.UpdatedAt = report.UpdatedAt.Add(time.Second)
	report.Network.TotalUp = 1
	if err := clients.ObserveTraffic(report); err != nil {
		t.Fatal(err)
	}
	stale := time.Now().UTC().Add(-3 * time.Minute)
	if err := db.Model(&models.Client{}).Where("uuid = ?", id).Update("traffic_lifetime_at", stale).Error; err != nil {
		t.Fatal(err)
	}
	if _, e := adminRebaseClientTrafficCounters(context.Background(), req); e == nil {
		t.Fatal("stale snapshot accepted recovery")
	}
	if _, e := adminRebaseClientTrafficCounters(context.Background(), &rpc.JsonRpcRequest{Params: map[string]any{}}); e == nil {
		t.Fatal("missing node accepted")
	}
}

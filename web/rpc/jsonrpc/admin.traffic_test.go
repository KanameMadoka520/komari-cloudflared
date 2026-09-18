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
	report := v2.Report{UUID: id, UpdatedAt: time.Now().UTC(), Network: v2.NetworkReport{TotalUp: 1000, TotalDown: 2000}}
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

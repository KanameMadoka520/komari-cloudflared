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

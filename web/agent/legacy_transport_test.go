package agent

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	v2 "github.com/komari-monitor/komari/protocol/v2"
	"github.com/komari-monitor/komari/web/connection"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestPingUsesAgentWireProtocol(t *testing.T) {
	for _, modern := range []bool{false, true} {
		name := "legacy"
		if modern {
			name = "v2"
		}
		t.Run(name, func(t *testing.T) {
			ready := make(chan *connection.SafeConn, 1)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				upgrader := websocket.Upgrader{}
				conn, err := upgrader.Upgrade(w, r, nil)
				if err != nil {
					return
				}
				ready <- connection.NewSafeConn(conn)
			}))
			defer server.Close()
			receiver, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), nil)
			if err != nil {
				t.Fatal(err)
			}
			defer receiver.Close()
			sender := <-ready
			defer sender.Close()
			uuid := "wire-test-" + name
			SetConnectedClients(uuid, sender)
			SetClientProtocol(uuid, modern)
			defer DeleteConnectedClients(uuid)
			if !modern && DispatchV2Event(uuid, v2.MethodAgentFile, map[string]any{}) {
				t.Fatal("v2 file request sent to legacy agent")
			}
			if !DispatchPing(uuid, v2.PingParams{TaskID: 7, Type: "icmp", Target: "127.0.0.1"}) {
				t.Fatal("dispatch failed")
			}
			receiver.SetReadDeadline(time.Now().Add(time.Second))
			var payload map[string]json.RawMessage
			if err := receiver.ReadJSON(&payload); err != nil {
				t.Fatal(err)
			}
			if modern {
				if string(payload["method"]) != `"agent.ping"` || payload["params"] == nil {
					t.Fatalf("wrong v2 envelope: %s", payload)
				}
			} else if string(payload["message"]) != `"ping"` || string(payload["ping_task_id"]) != "7" || payload["jsonrpc"] != nil {
				t.Fatalf("wrong legacy envelope: %s", payload)
			}
		})
	}
}

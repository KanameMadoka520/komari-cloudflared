package client

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	v2 "github.com/komari-monitor/komari/protocol/v2"
	agent_runtime "github.com/komari-monitor/komari/web/agent"
	"github.com/komari-monitor/komari/web/api"
	"github.com/komari-monitor/komari/web/connection"
	"net/http"
	"time"
)

// Legacy payloads share v2's report shape. Identity always comes from the
// authenticated request, never the UUID supplied in the report body.
func UploadReport(c *gin.Context) {
	uuid, ok := clientUUIDFromContext(c)
	if !ok {
		api.RespondError(c, http.StatusUnauthorized, "Invalid client token")
		return
	}
	var report v2.Report
	if err := c.ShouldBindJSON(&report); err != nil {
		api.RespondError(c, http.StatusBadRequest, "Invalid report")
		return
	}
	if err := ingestReportWithProtocol(uuid, report, true, false); err != nil {
		api.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	api.RespondSuccess(c, nil)
}

func UploadBasicInfo(c *gin.Context) {
	uuid, ok := clientUUIDFromContext(c)
	if !ok {
		api.RespondError(c, http.StatusUnauthorized, "Invalid client token")
		return
	}
	var info map[string]interface{}
	if err := c.ShouldBindJSON(&info); err != nil {
		api.RespondError(c, http.StatusBadRequest, "Invalid basic info")
		return
	}
	if err := ingestBasicInfo(uuid, info, c.ClientIP()); err != nil {
		api.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	api.RespondSuccess(c, nil)
}

func WebSocketReport(c *gin.Context) {
	uuid, ok := clientUUIDFromContext(c)
	if !ok {
		api.RespondError(c, http.StatusUnauthorized, "Invalid client token")
		return
	}
	conn, err := api.UpgradeSafeConn(c)
	if err != nil {
		return
	}
	defer conn.Close()
	if old := agent_runtime.GetConnectedClients()[uuid]; old != nil {
		go old.Close()
	}
	agent_runtime.SetClientProtocol(uuid, false)
	agent_runtime.SetConnectedClients(uuid, conn)
	go notifierOnline(uuid, conn.ID)
	defer func() { agent_runtime.DeleteClientConditionally(uuid, conn); notifierOffline(uuid, conn.ID) }()
	for {
		conn.SetReadDeadline(time.Now().Add(readWait))
		_, message, err := conn.ReadMessage()
		if err != nil {
			return
		}
		processLegacyMessage(conn, message, uuid)
	}
}

func processLegacyMessage(conn *connection.SafeConn, message []byte, uuid string) {
	var envelope struct {
		Type   string `json:"type"`
		TaskID uint   `json:"task_id"`
		Value  int    `json:"value"`
	}
	if err := json.Unmarshal(message, &envelope); err != nil {
		conn.WriteJSON(gin.H{"status": "error", "error": "Invalid JSON"})
		return
	}
	var err error
	switch envelope.Type {
	case "", "report":
		var report v2.Report
		if err = json.Unmarshal(message, &report); err == nil {
			err = ingestReportWithProtocol(uuid, report, false, false)
		}
	case "ping_result":
		err = ingestPingResult(uuid, envelope.TaskID, envelope.Value)
	default:
		conn.WriteJSON(gin.H{"status": "error", "error": "Unknown message type"})
		return
	}
	if err != nil {
		conn.WriteJSON(gin.H{"status": "error", "error": err.Error()})
	}
}

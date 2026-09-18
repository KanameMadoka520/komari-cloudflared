package jsonrpc

import (
	"context"
	"fmt"
	"time"

	"github.com/komari-monitor/komari/database/auditlog"
	"github.com/komari-monitor/komari/database/clients"
	"github.com/komari-monitor/komari/database/models"
	"github.com/komari-monitor/komari/database/records"
	"github.com/komari-monitor/komari/internal/metricstore"
	"github.com/komari-monitor/komari/pkg/rpc"
	agent_runtime "github.com/komari-monitor/komari/web/agent"
)

// admin.client.go
// client 资源的 RPC2 方法（admin 命名空间）。承载原 web/api/admin/client.go 的业务逻辑，
// 包含审计日志与运行时副作用。传统 REST handler 经 CallFromGin 转调这些方法。

func init() {
	RegisterWithGroupAndMeta("addClient", rpc.RoleAdmin, adminAddClient, &rpc.MethodMeta{
		Name:    "admin:addClient",
		Summary: "Create a new client",
		Params: []rpc.ParamMeta{
			{Name: "name", Type: "string", Required: false, Description: "Optional client name"},
		},
		Returns: "{ uuid: string, token: string }",
	})
	RegisterWithGroupAndMeta("editClient", rpc.RoleAdmin, adminEditClient, &rpc.MethodMeta{
		Name:    "admin:editClient",
		Summary: "Edit a client (partial update)",
		Params: []rpc.ParamMeta{
			{Name: "uuid", Type: "string", Required: true, Description: "Client UUID"},
		},
		Returns: "null",
	})
	RegisterWithGroupAndMeta("removeClient", rpc.RoleAdmin, adminRemoveClient, &rpc.MethodMeta{
		Name:    "admin:removeClient",
		Summary: "Delete a client",
		Params: []rpc.ParamMeta{
			{Name: "uuid", Type: "string", Required: true, Description: "Client UUID"},
		},
		Returns: "null",
	})
	RegisterWithGroupAndMeta("getClient", rpc.RoleAdmin, adminGetClient, &rpc.MethodMeta{
		Name:    "admin:getClient",
		Summary: "Get a client by UUID",
		Params: []rpc.ParamMeta{
			{Name: "uuid", Type: "string", Required: true, Description: "Client UUID"},
		},
		Returns: "Client",
	})
	RegisterWithGroupAndMeta("listClients", rpc.RoleAdmin, adminListClients, &rpc.MethodMeta{
		Name:    "admin:listClients",
		Summary: "List all clients (basic info)",
		Returns: "Client[]",
	})
	RegisterWithGroupAndMeta("getClientToken", rpc.RoleAdmin, adminGetClientToken, &rpc.MethodMeta{
		Name:    "admin:getClientToken",
		Summary: "Get a client's token by UUID",
		Params: []rpc.ParamMeta{
			{Name: "uuid", Type: "string", Required: true, Description: "Client UUID"},
		},
		Returns: "{ token: string }",
	})
	RegisterWithGroupAndMeta("clearRecords", rpc.RoleAdmin, adminClearRecords, &rpc.MethodMeta{
		Name:    "admin:clearRecords",
		Summary: "Delete all load records",
		Returns: "null",
	})
	RegisterWithGroupAndMeta("getClientTraffic", rpc.RoleAdmin, adminGetClientTraffic, &rpc.MethodMeta{
		Name:    "admin:getClientTraffic",
		Summary: "Get a client's current traffic-cycle usage",
		Params:  []rpc.ParamMeta{{Name: "uuid", Type: "string", Required: true, Description: "Client UUID"}},
		Returns: "{ upload: number, download: number, total: number, total_mode: boolean, billing_type: string, reset_at: string, raw_upload: number, raw_download: number }",
	})
	RegisterWithGroupAndMeta("resetClientTraffic", rpc.RoleAdmin, adminResetClientTraffic, &rpc.MethodMeta{
		Name:    "admin:resetClientTraffic",
		Summary: "Start a new zeroed traffic cycle for a client",
		Params:  []rpc.ParamMeta{{Name: "uuid", Type: "string", Required: true, Description: "Client UUID"}},
		Returns: "null",
	})
	RegisterWithGroupAndMeta("setClientTrafficUsage", rpc.RoleAdmin, adminSetClientTrafficUsage, &rpc.MethodMeta{
		Name:    "admin:setClientTrafficUsage",
		Summary: "Start a traffic cycle with manually entered usage",
		Params: []rpc.ParamMeta{
			{Name: "uuid", Type: "string", Required: true, Description: "Client UUID"},
			{Name: "total", Type: "number", Required: false, Description: "Provider usage in bytes; exclusive with upload/download"},
			{Name: "upload", Type: "number", Required: false, Description: "Current upload usage in bytes; requires download"},
			{Name: "download", Type: "number", Required: false, Description: "Current download usage in bytes; requires upload"},
		},
		Returns: "null",
	})
}

// auditActor 从上下文提取审计用的 actor UUID 与来源 IP。
func auditActor(ctx context.Context) (uuid, ip string) {
	if meta := rpc.MetaFromContext(ctx); meta != nil {
		uuid = meta.UserUUID
		ip = meta.RemoteIP
	}
	return uuid, ip
}

func adminAddClient(ctx context.Context, req *rpc.JsonRpcRequest) (any, *rpc.JsonRpcError) {
	var params struct {
		Name string `json:"name"`
	}
	req.BindParams(&params)

	var (
		uuid, token string
		err         error
	)
	if params.Name == "" {
		uuid, token, err = clients.CreateClient()
	} else {
		uuid, token, err = clients.CreateClientWithName(params.Name)
	}
	if err != nil {
		return nil, rpc.MakeError(rpc.InternalError, err.Error(), nil)
	}
	if params.Name != "" {
		actor, ip := auditActor(ctx)
		auditlog.Log(ip, actor, "create client:"+uuid, "info")
	}
	return map[string]any{"uuid": uuid, "token": token}, nil
}

func adminEditClient(ctx context.Context, req *rpc.JsonRpcRequest) (any, *rpc.JsonRpcError) {
	var update map[string]interface{}
	if err := req.BindParams(&update); err != nil || update == nil {
		return nil, rpc.MakeError(rpc.InvalidParams, "Invalid params", nil)
	}
	uuid, _ := update["uuid"].(string)
	if uuid == "" {
		return nil, rpc.MakeError(rpc.InvalidParams, "Invalid or missing UUID", nil)
	}
	if err := clients.SaveClient(update); err != nil {
		return nil, rpc.MakeError(rpc.InternalError, err.Error(), nil)
	}
	actor, ip := auditActor(ctx)
	auditlog.Log(ip, actor, "edit client:"+uuid, "info")
	return nil, nil
}

func adminRemoveClient(ctx context.Context, req *rpc.JsonRpcRequest) (any, *rpc.JsonRpcError) {
	var params struct {
		UUID string `json:"uuid"`
	}
	req.BindParams(&params)
	if params.UUID == "" {
		return nil, rpc.MakeError(rpc.InvalidParams, "Invalid or missing UUID", nil)
	}
	if err := clients.DeleteClient(params.UUID); err != nil {
		return nil, rpc.MakeError(rpc.InternalError, "Failed to delete client"+err.Error(), nil)
	}
	metricstore.DeleteEntityAsync(params.UUID)
	actor, ip := auditActor(ctx)
	auditlog.Log(ip, actor, "delete client:"+params.UUID, "warn")
	agent_runtime.DeleteConnectedClients(params.UUID)
	agent_runtime.DeleteLatestReport(params.UUID)
	return nil, nil
}

func adminGetClient(_ context.Context, req *rpc.JsonRpcRequest) (any, *rpc.JsonRpcError) {
	var params struct {
		UUID string `json:"uuid"`
	}
	req.BindParams(&params)
	if params.UUID == "" {
		return nil, rpc.MakeError(rpc.InvalidParams, "Invalid or missing UUID", nil)
	}
	result, err := clients.GetClientByUUID(params.UUID)
	if err != nil {
		return nil, rpc.MakeError(rpc.InternalError, err.Error(), nil)
	}
	return result, nil
}

func adminListClients(_ context.Context, _ *rpc.JsonRpcRequest) (any, *rpc.JsonRpcError) {
	cls, err := clients.GetAllClientBasicInfo()
	if err != nil {
		return nil, rpc.MakeError(rpc.InternalError, err.Error(), nil)
	}
	return cls, nil
}

func currentTrafficCounters(ctx context.Context, uuid string) (int64, int64, error) {
	if report := agent_runtime.GetLatestReport()[uuid]; report != nil {
		return report.Network.TotalUp, report.Network.TotalDown, nil
	}
	latest, err := metricstore.GetLatestTrafficBefore(ctx, []string{uuid}, time.Now().UTC().Add(time.Second))
	if err != nil {
		return 0, 0, err
	}
	if record, ok := latest[uuid]; ok {
		return record.NetTotalUp, record.NetTotalDown, nil
	}
	return 0, 0, fmt.Errorf("no traffic counter available; wait for the agent to report before adjusting usage")
}

func adminGetClientTraffic(ctx context.Context, req *rpc.JsonRpcRequest) (any, *rpc.JsonRpcError) {
	clients.TrafficMu.Lock()
	defer clients.TrafficMu.Unlock()
	var params struct {
		UUID string `json:"uuid"`
	}
	if err := req.BindParams(&params); err != nil || params.UUID == "" {
		return nil, rpc.MakeError(rpc.InvalidParams, "Invalid or missing UUID", nil)
	}
	client, err := clients.GetClientByUUID(params.UUID)
	if err != nil {
		return nil, rpc.MakeError(rpc.InvalidParams, "Client not found", params.UUID)
	}
	rawUp, rawDown, err := currentTrafficCounters(ctx, params.UUID)
	if err != nil {
		return nil, rpc.MakeError(rpc.InternalError, "Failed to read traffic counters", err.Error())
	}

	return trafficUsage(client, rawUp, rawDown), nil
}

func trafficUsage(client models.Client, rawUp, rawDown int64) map[string]any {
	usedUp, usedDown := clients.EffectiveTraffic(client, rawUp, rawDown)
	return map[string]any{
		"upload": usedUp, "download": usedDown,
		"total":        clients.EffectiveTrafficTotal(client, rawUp, rawDown),
		"total_mode":   client.TrafficInitialTotal != nil,
		"billing_type": client.TrafficLimitType,
		"raw_upload":   rawUp, "raw_download": rawDown,
		"reset_at":       client.TrafficResetAt,
		"initial_upload": client.TrafficInitialUp, "initial_download": client.TrafficInitialDown,
		"initial_total": client.TrafficInitialTotal,
	}
}

func setClientTrafficCycle(ctx context.Context, uuid string, initialUp, initialDown int64, total *int64) (any, *rpc.JsonRpcError) {
	clients.TrafficMu.Lock()
	defer clients.TrafficMu.Unlock()
	if uuid == "" || initialUp < 0 || initialDown < 0 || initialUp > 9007199254740991 || initialDown > 9007199254740991 ||
		(total != nil && (*total < 0 || *total > 9007199254740991)) {
		return nil, rpc.MakeError(rpc.InvalidParams, "UUID is required; usage must be a non-negative safe integer", nil)
	}
	client, err := clients.GetClientByUUID(uuid)
	if err != nil {
		return nil, rpc.MakeError(rpc.InvalidParams, "Client not found", nil)
	}
	rawUp, rawDown, err := currentTrafficCounters(ctx, uuid)
	if err != nil {
		return nil, rpc.MakeError(rpc.InternalError, "Failed to read traffic counters", err.Error())
	}
	at := time.Now().UTC()
	if err := clients.SetTrafficUsageBaseline(uuid, rawUp, rawDown, initialUp, initialDown, total, at); err != nil {
		return nil, rpc.MakeError(rpc.InternalError, "Failed to save traffic cycle", err.Error())
	}
	client.TrafficResetAt, client.TrafficObservedAt = &at, &at
	client.TrafficInitialUp, client.TrafficInitialDown = initialUp, initialDown
	client.TrafficUsedUp, client.TrafficUsedDown = initialUp, initialDown
	client.TrafficInitialTotal = total
	return trafficUsage(client, rawUp, rawDown), nil
}

func adminResetClientTraffic(ctx context.Context, req *rpc.JsonRpcRequest) (any, *rpc.JsonRpcError) {
	var params struct {
		UUID string `json:"uuid"`
	}
	if err := req.BindParams(&params); err != nil {
		return nil, rpc.MakeError(rpc.InvalidParams, "Invalid params", nil)
	}
	result, rpcErr := setClientTrafficCycle(ctx, params.UUID, 0, 0, nil)
	if rpcErr != nil {
		return nil, rpcErr
	}
	actor, ip := auditActor(ctx)
	auditlog.Log(ip, actor, "reset traffic cycle:"+params.UUID, "warn")
	return result, nil
}

func adminSetClientTrafficUsage(ctx context.Context, req *rpc.JsonRpcRequest) (any, *rpc.JsonRpcError) {
	var params struct {
		UUID     string `json:"uuid"`
		Upload   *int64 `json:"upload"`
		Download *int64 `json:"download"`
		Total    *int64 `json:"total"`
	}
	if err := req.BindParams(&params); err != nil {
		return nil, rpc.MakeError(rpc.InvalidParams, "Invalid params", nil)
	}
	// Check presence as well as values: even null mixed fields are ambiguous.
	var fields map[string]any
	if err := req.BindParams(&fields); err != nil {
		return nil, rpc.MakeError(rpc.InvalidParams, "Invalid params", nil)
	}
	_, hasTotal := fields["total"]
	_, hasUp := fields["upload"]
	_, hasDown := fields["download"]
	var up, down int64
	if hasTotal {
		if params.Total == nil || hasUp || hasDown {
			return nil, rpc.MakeError(rpc.InvalidParams, "Provide total alone, or both upload and download", nil)
		}
	} else {
		if params.Upload == nil || params.Download == nil {
			return nil, rpc.MakeError(rpc.InvalidParams, "Provide total alone, or both upload and download", nil)
		}
		up, down = *params.Upload, *params.Download
	}
	result, rpcErr := setClientTrafficCycle(ctx, params.UUID, up, down, params.Total)
	if rpcErr != nil {
		return nil, rpcErr
	}
	actor, ip := auditActor(ctx)
	auditlog.Log(ip, actor, "set traffic cycle:"+params.UUID, "warn")
	return result, nil
}

func adminGetClientToken(_ context.Context, req *rpc.JsonRpcRequest) (any, *rpc.JsonRpcError) {
	var params struct {
		UUID string `json:"uuid"`
	}
	req.BindParams(&params)
	if params.UUID == "" {
		return nil, rpc.MakeError(rpc.InvalidParams, "Invalid or missing UUID", nil)
	}
	token, err := clients.GetClientTokenByUUID(params.UUID)
	if err != nil {
		return nil, rpc.MakeError(rpc.InternalError, err.Error(), nil)
	}
	return map[string]any{"token": token}, nil
}

func adminClearRecords(ctx context.Context, _ *rpc.JsonRpcRequest) (any, *rpc.JsonRpcError) {
	if err := records.DeleteAll(); err != nil {
		return nil, rpc.MakeError(rpc.InternalError, "Failed to delete Record"+err.Error(), nil)
	}
	actor, ip := auditActor(ctx)
	auditlog.Log(ip, actor, "clear records", "warn")
	return nil, nil
}

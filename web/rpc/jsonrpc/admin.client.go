package jsonrpc

import (
	"context"
	"time"

	"github.com/komari-monitor/komari/database/auditlog"
	"github.com/komari-monitor/komari/database/clients"
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
		Returns: "{ upload: number, download: number, raw_upload: number, raw_download: number }",
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
			{Name: "upload", Type: "number", Required: true, Description: "Current upload usage in bytes"},
			{Name: "download", Type: "number", Required: true, Description: "Current download usage in bytes"},
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
	return 0, 0, nil
}

func adminGetClientTraffic(ctx context.Context, req *rpc.JsonRpcRequest) (any, *rpc.JsonRpcError) {
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
	usedUp, usedDown := clients.EffectiveTraffic(client, rawUp, rawDown)
	return map[string]any{
		"upload":           usedUp,
		"download":         usedDown,
		"raw_upload":       rawUp,
		"raw_download":     rawDown,
		"reset_at":         client.TrafficResetAt,
		"initial_upload":   client.TrafficInitialUp,
		"initial_download": client.TrafficInitialDown,
	}, nil
}

func setClientTrafficCycle(ctx context.Context, uuid string, initialUp, initialDown int64) *rpc.JsonRpcError {
	if uuid == "" || initialUp < 0 || initialDown < 0 {
		return rpc.MakeError(rpc.InvalidParams, "UUID and traffic usage must be non-negative", nil)
	}
	rawUp, rawDown, err := currentTrafficCounters(ctx, uuid)
	if err != nil {
		return rpc.MakeError(rpc.InternalError, "Failed to read traffic counters", err.Error())
	}
	if err := clients.SetTrafficBaseline(uuid, rawUp, rawDown, initialUp, initialDown, time.Now().UTC()); err != nil {
		return rpc.MakeError(rpc.InternalError, "Failed to save traffic cycle", err.Error())
	}
	return nil
}

func adminResetClientTraffic(ctx context.Context, req *rpc.JsonRpcRequest) (any, *rpc.JsonRpcError) {
	var params struct {
		UUID string `json:"uuid"`
	}
	if err := req.BindParams(&params); err != nil {
		return nil, rpc.MakeError(rpc.InvalidParams, "Invalid params", nil)
	}
	if rpcErr := setClientTrafficCycle(ctx, params.UUID, 0, 0); rpcErr != nil {
		return nil, rpcErr
	}
	actor, ip := auditActor(ctx)
	auditlog.Log(ip, actor, "reset traffic cycle:"+params.UUID, "warn")
	return nil, nil
}

func adminSetClientTrafficUsage(ctx context.Context, req *rpc.JsonRpcRequest) (any, *rpc.JsonRpcError) {
	var params struct {
		UUID     string `json:"uuid"`
		Upload   int64  `json:"upload"`
		Download int64  `json:"download"`
	}
	if err := req.BindParams(&params); err != nil {
		return nil, rpc.MakeError(rpc.InvalidParams, "Invalid params", nil)
	}
	if rpcErr := setClientTrafficCycle(ctx, params.UUID, params.Upload, params.Download); rpcErr != nil {
		return nil, rpcErr
	}
	actor, ip := auditActor(ctx)
	auditlog.Log(ip, actor, "set traffic cycle:"+params.UUID, "warn")
	return nil, nil
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

// atlhyper_master_v2/gateway/handler/ops.go
// Operations API Handler
// 便捷操作接口，封装 CommandService
package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/atlhyper_master_v2/mq"
	"AtlHyper/atlhyper_master_v2/service"
	"AtlHyper/atlhyper_master_v2/service/operations"
)

// OpsHandler 操作 Handler
type OpsHandler struct {
	svc service.Service
	bus mq.CommandBus
}

// NewOpsHandler 创建 OpsHandler
func NewOpsHandler(svc service.Service, bus mq.CommandBus) *OpsHandler {
	return &OpsHandler{
		svc: svc,
		bus: bus,
	}
}

// ==================== 请求结构 ====================

// PodLogsRequest Pod 日志请求
type PodLogsRequest struct {
	ClusterID string `json:"cluster_id"`
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	Container string `json:"container,omitempty"`
	TailLines int    `json:"tail_lines,omitempty"`
}

// PodRestartRequest Pod 重启请求
type PodRestartRequest struct {
	ClusterID string `json:"cluster_id"`
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

// DeploymentScaleRequest 扩缩容请求
type DeploymentScaleRequest struct {
	ClusterID string `json:"cluster_id"`
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	Replicas  int    `json:"replicas"`
}

// DeploymentRestartRequest 滚动重启请求
type DeploymentRestartRequest struct {
	ClusterID string `json:"cluster_id"`
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

// DeploymentImageRequest 更新镜像请求
type DeploymentImageRequest struct {
	ClusterID string `json:"cluster_id"`
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	Container string `json:"container"`
	Image     string `json:"image"`
}

// NodeCordonRequest Node 封锁/解封请求
type NodeCordonRequest struct {
	ClusterID string `json:"cluster_id"`
	Name      string `json:"name"`
}

// ConfigMapDataRequest 获取 ConfigMap 数据请求
type ConfigMapDataRequest struct {
	ClusterID string `json:"cluster_id"`
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

// SecretDataRequest 获取 Secret 数据请求
type SecretDataRequest struct {
	ClusterID string `json:"cluster_id"`
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

// ==================== Handler 方法 ====================

// PodLogs 获取 Pod 日志（同步等待）
// POST /api/v2/ops/pods/logs
func (h *OpsHandler) PodLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req PodLogsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求参数无效")
		return
	}

	if req.ClusterID == "" || req.Namespace == "" || req.Name == "" {
		writeError(w, http.StatusBadRequest, "cluster_id, namespace, name 不能为空")
		return
	}

	tailLines := req.TailLines
	if tailLines <= 0 {
		tailLines = 100
	}

	params := map[string]interface{}{
		"tailLines": tailLines,
	}
	if req.Container != "" {
		params["container"] = req.Container
	}

	// 1. 创建指令
	resp, err := h.svc.CreateCommand(&operations.CreateCommandRequest{
		ClusterID:       req.ClusterID,
		Action:          model.ActionGetLogs,
		TargetKind:      "Pod",
		TargetNamespace: req.Namespace,
		TargetName:      req.Name,
		Params:          params,
		Source:          "web",
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "创建指令失败: "+err.Error())
		return
	}

	// 2. 同步等待结果（30秒超时）
	result, err := h.bus.WaitCommandResult(resp.CommandID, 30*time.Second)
	if err != nil {
		writeError(w, http.StatusGatewayTimeout, "获取日志超时")
		return
	}
	if result == nil {
		writeError(w, http.StatusGatewayTimeout, "获取日志超时，Agent 可能未响应")
		return
	}

	// 3. 检查执行结果
	if !result.Success {
		writeError(w, http.StatusInternalServerError, result.Error)
		return
	}

	// 4. 返回日志
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "获取成功",
		"data": map[string]string{
			"logs": result.Output,
		},
	})
}

// PodRestart 重启 Pod（通过删除触发重建）
// POST /api/v2/ops/pods/restart
func (h *OpsHandler) PodRestart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req PodRestartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求参数无效")
		return
	}

	if req.ClusterID == "" || req.Namespace == "" || req.Name == "" {
		writeError(w, http.StatusBadRequest, "cluster_id, namespace, name 不能为空")
		return
	}

	resp, err := h.svc.CreateCommand(&operations.CreateCommandRequest{
		ClusterID:       req.ClusterID,
		Action:          model.ActionDelete,
		TargetKind:      "Pod",
		TargetNamespace: req.Namespace,
		TargetName:      req.Name,
		Source:          "web",
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "创建指令失败: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message":    "Pod 重启指令已下发",
		"command_id": resp.CommandID,
		"status":     resp.Status,
	})
}

// DeploymentScale 扩缩容
// POST /api/v2/ops/deployments/scale
func (h *OpsHandler) DeploymentScale(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req DeploymentScaleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求参数无效")
		return
	}

	if req.ClusterID == "" || req.Namespace == "" || req.Name == "" {
		writeError(w, http.StatusBadRequest, "cluster_id, namespace, name 不能为空")
		return
	}

	if req.Replicas < 0 {
		writeError(w, http.StatusBadRequest, "replicas 不能为负数")
		return
	}

	resp, err := h.svc.CreateCommand(&operations.CreateCommandRequest{
		ClusterID:       req.ClusterID,
		Action:          model.ActionScale,
		TargetKind:      "Deployment",
		TargetNamespace: req.Namespace,
		TargetName:      req.Name,
		Params: map[string]interface{}{
			"replicas": req.Replicas,
		},
		Source: "web",
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "创建指令失败: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message":    "扩缩容指令已下发",
		"command_id": resp.CommandID,
		"status":     resp.Status,
	})
}

// DeploymentRestart 滚动重启 Deployment
// POST /api/v2/ops/deployments/restart
func (h *OpsHandler) DeploymentRestart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req DeploymentRestartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求参数无效")
		return
	}

	if req.ClusterID == "" || req.Namespace == "" || req.Name == "" {
		writeError(w, http.StatusBadRequest, "cluster_id, namespace, name 不能为空")
		return
	}

	resp, err := h.svc.CreateCommand(&operations.CreateCommandRequest{
		ClusterID:       req.ClusterID,
		Action:          model.ActionRestart,
		TargetKind:      "Deployment",
		TargetNamespace: req.Namespace,
		TargetName:      req.Name,
		Source:          "web",
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "创建指令失败: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message":    "重启指令已下发",
		"command_id": resp.CommandID,
		"status":     resp.Status,
	})
}

// DeploymentImage 更新 Deployment 镜像
// POST /api/v2/ops/deployments/image
func (h *OpsHandler) DeploymentImage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req DeploymentImageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求参数无效")
		return
	}

	if req.ClusterID == "" || req.Namespace == "" || req.Name == "" {
		writeError(w, http.StatusBadRequest, "cluster_id, namespace, name 不能为空")
		return
	}

	if req.Container == "" || req.Image == "" {
		writeError(w, http.StatusBadRequest, "container, image 不能为空")
		return
	}

	resp, err := h.svc.CreateCommand(&operations.CreateCommandRequest{
		ClusterID:       req.ClusterID,
		Action:          model.ActionUpdateImage,
		TargetKind:      "Deployment",
		TargetNamespace: req.Namespace,
		TargetName:      req.Name,
		Params: map[string]interface{}{
			"container": req.Container,
			"image":     req.Image,
		},
		Source: "web",
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "创建指令失败: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message":    "镜像更新指令已下发",
		"command_id": resp.CommandID,
		"status":     resp.Status,
	})
}

// NodeCordon 封锁 Node
// POST /api/v2/ops/nodes/cordon
func (h *OpsHandler) NodeCordon(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req NodeCordonRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求参数无效")
		return
	}

	if req.ClusterID == "" || req.Name == "" {
		writeError(w, http.StatusBadRequest, "cluster_id, name 不能为空")
		return
	}

	resp, err := h.svc.CreateCommand(&operations.CreateCommandRequest{
		ClusterID:  req.ClusterID,
		Action:     model.ActionCordon,
		TargetKind: "Node",
		TargetName: req.Name,
		Source:     "web",
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "创建指令失败: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message":    "Node 封锁指令已下发",
		"command_id": resp.CommandID,
		"status":     resp.Status,
	})
}

// NodeUncordon 解封 Node
// POST /api/v2/ops/nodes/uncordon
func (h *OpsHandler) NodeUncordon(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req NodeCordonRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求参数无效")
		return
	}

	if req.ClusterID == "" || req.Name == "" {
		writeError(w, http.StatusBadRequest, "cluster_id, name 不能为空")
		return
	}

	resp, err := h.svc.CreateCommand(&operations.CreateCommandRequest{
		ClusterID:  req.ClusterID,
		Action:     model.ActionUncordon,
		TargetKind: "Node",
		TargetName: req.Name,
		Source:     "web",
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "创建指令失败: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message":    "Node 解封指令已下发",
		"command_id": resp.CommandID,
		"status":     resp.Status,
	})
}

// ConfigMapData 获取 ConfigMap 数据（同步等待）
// POST /api/v2/ops/configmaps/data
func (h *OpsHandler) ConfigMapData(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req ConfigMapDataRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求参数无效")
		return
	}

	if req.ClusterID == "" || req.Namespace == "" || req.Name == "" {
		writeError(w, http.StatusBadRequest, "cluster_id, namespace, name 不能为空")
		return
	}

	// 1. 创建指令
	resp, err := h.svc.CreateCommand(&operations.CreateCommandRequest{
		ClusterID:       req.ClusterID,
		Action:          model.ActionGetConfigMap,
		TargetKind:      "ConfigMap",
		TargetNamespace: req.Namespace,
		TargetName:      req.Name,
		Source:          "web",
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "创建指令失败: "+err.Error())
		return
	}

	// 2. 同步等待结果（30秒超时）
	result, err := h.bus.WaitCommandResult(resp.CommandID, 30*time.Second)
	if err != nil {
		writeError(w, http.StatusGatewayTimeout, "获取数据超时")
		return
	}
	if result == nil {
		writeError(w, http.StatusGatewayTimeout, "获取数据超时，Agent 可能未响应")
		return
	}

	// 3. 检查执行结果
	if !result.Success {
		writeError(w, http.StatusInternalServerError, result.Error)
		return
	}

	// 4. 返回数据
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "获取成功",
		"data":    result.Output,
	})
}

// SecretData 获取 Secret 数据（同步等待）
// POST /api/v2/ops/secrets/data
func (h *OpsHandler) SecretData(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req SecretDataRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求参数无效")
		return
	}

	if req.ClusterID == "" || req.Namespace == "" || req.Name == "" {
		writeError(w, http.StatusBadRequest, "cluster_id, namespace, name 不能为空")
		return
	}

	// 1. 创建指令
	resp, err := h.svc.CreateCommand(&operations.CreateCommandRequest{
		ClusterID:       req.ClusterID,
		Action:          model.ActionGetSecret,
		TargetKind:      "Secret",
		TargetNamespace: req.Namespace,
		TargetName:      req.Name,
		Source:          "web",
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "创建指令失败: "+err.Error())
		return
	}

	// 2. 同步等待结果（30秒超时）
	result, err := h.bus.WaitCommandResult(resp.CommandID, 30*time.Second)
	if err != nil {
		writeError(w, http.StatusGatewayTimeout, "获取数据超时")
		return
	}
	if result == nil {
		writeError(w, http.StatusGatewayTimeout, "获取数据超时，Agent 可能未响应")
		return
	}

	// 3. 检查执行结果
	if !result.Success {
		writeError(w, http.StatusInternalServerError, result.Error)
		return
	}

	// 4. 返回数据
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "获取成功",
		"data":    result.Output,
	})
}

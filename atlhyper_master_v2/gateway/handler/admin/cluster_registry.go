// atlhyper_master_v2/gateway/handler/admin/cluster_registry.go
// 集群注册表管理 Handler：列表、登记、改显示名与描述、签发令牌、删除登记
package admin

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/gateway/handler"
	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/common/logger"
)

var clusterRegistryLog = logger.Module("ClusterRegistryHandler")

// clusterIDPathParam 路由里的路径参数名（见 gateway/routes.go 的 {clusterId}）
const clusterIDPathParam = "clusterId"

// ClusterRegistryQuery / ClusterRegistryOps 本 handler 所需的最小 service 能力
type ClusterRegistryQuery interface {
	ListClusterRegistry(ctx context.Context) ([]model.ClusterRegistryItem, error)
	GetClusterRegistryItem(ctx context.Context, clusterID string) (*model.ClusterRegistryItem, error)
}

type ClusterRegistryOps interface {
	RegisterCluster(ctx context.Context, req *model.RegisterClusterRequest) error
	UpdateClusterProfile(ctx context.Context, clusterID string, req *model.UpdateClusterProfileRequest) error
	IssueClusterToken(ctx context.Context, clusterID string) (*model.ClusterTokenIssued, error)
	UnregisterCluster(ctx context.Context, clusterID string) error
}

// ClusterRegistryHandler 集群注册表管理 Handler
type ClusterRegistryHandler struct {
	querySvc ClusterRegistryQuery
	opsSvc   ClusterRegistryOps
}

// NewClusterRegistryHandler 创建 ClusterRegistryHandler
func NewClusterRegistryHandler(querySvc ClusterRegistryQuery, opsSvc ClusterRegistryOps) *ClusterRegistryHandler {
	return &ClusterRegistryHandler{querySvc: querySvc, opsSvc: opsSvc}
}

// List 集群管理页的全部行（已登记的 + 正在上报但未登记的）
// GET /api/v2/admin/clusters
func (h *ClusterRegistryHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		handler.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	items, err := h.querySvc.ListClusterRegistry(r.Context())
	if err != nil {
		clusterRegistryLog.Error("获取集群列表失败", "err", err)
		handler.WriteError(w, http.StatusInternalServerError, "获取集群列表失败")
		return
	}
	if items == nil {
		items = []model.ClusterRegistryItem{}
	}

	handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message": "获取成功",
		"data":    items,
		"total":   len(items),
	})
}

// Register 登记集群
// POST /api/v2/admin/clusters
func (h *ClusterRegistryHandler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		handler.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req model.RegisterClusterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handler.WriteError(w, http.StatusBadRequest, "请求参数无效")
		return
	}

	if err := h.opsSvc.RegisterCluster(r.Context(), &req); err != nil {
		h.writeOpsError(w, "登记集群失败", err)
		return
	}

	handler.WriteJSON(w, http.StatusCreated, h.withCurrentRow(r.Context(), "已登记", req.ClusterID))
}

// UpdateProfile 修改显示名与描述
// PUT /api/v2/admin/clusters/{clusterId}
func (h *ClusterRegistryHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		handler.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	clusterID := r.PathValue(clusterIDPathParam)
	var req model.UpdateClusterProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handler.WriteError(w, http.StatusBadRequest, "请求参数无效")
		return
	}

	if err := h.opsSvc.UpdateClusterProfile(r.Context(), clusterID, &req); err != nil {
		h.writeOpsError(w, "修改集群信息失败", err)
		return
	}

	handler.WriteJSON(w, http.StatusOK, h.withCurrentRow(r.Context(), "已更新", clusterID))
}

// IssueToken 签发（或轮换）集群令牌。明文只在这一次响应里出现
// POST /api/v2/admin/clusters/{clusterId}/token
func (h *ClusterRegistryHandler) IssueToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		handler.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	issued, err := h.opsSvc.IssueClusterToken(r.Context(), r.PathValue(clusterIDPathParam))
	if err != nil {
		h.writeOpsError(w, "签发令牌失败", err)
		return
	}

	// 响应体里有令牌明文：不允许浏览器或中间代理缓存
	w.Header().Set("Cache-Control", "no-store")
	handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message": "已签发",
		"data":    issued,
	})
}

// Unregister 删除登记（令牌随之失效；不删除该集群的历史数据）
// DELETE /api/v2/admin/clusters/{clusterId}
func (h *ClusterRegistryHandler) Unregister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		handler.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	if err := h.opsSvc.UnregisterCluster(r.Context(), r.PathValue(clusterIDPathParam)); err != nil {
		h.writeOpsError(w, "删除集群登记失败", err)
		return
	}

	handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message": "已删除登记",
	})
}

// withCurrentRow 写操作成功后的响应：带上该集群最新的那一行，前端不必再拉一次列表。
//
// 读回失败不改变结果 —— 写入已经生效，这时报错会让调用方误以为没写进去；只是不带 data。
func (h *ClusterRegistryHandler) withCurrentRow(ctx context.Context, message, clusterID string) map[string]interface{} {
	response := map[string]interface{}{"message": message}
	item, err := h.querySvc.GetClusterRegistryItem(ctx, clusterID)
	if err != nil {
		clusterRegistryLog.Warn("写入成功，但读回集群信息失败", "cluster", clusterID, "err", err)
		return response
	}
	if item != nil {
		response["data"] = item
	}
	return response
}

// writeOpsError 把 service 返回的错误映射为状态码。
// 校验类错误的文案是写给管理员看的，原样返回；其余错误只记日志，不向外透露内部细节。
func (h *ClusterRegistryHandler) writeOpsError(w http.ResponseWriter, action string, err error) {
	switch {
	case errors.Is(err, model.ErrInvalidClusterInput):
		handler.WriteError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, database.ErrClusterExists):
		handler.WriteError(w, http.StatusConflict, "集群 ID 已登记")
	case errors.Is(err, database.ErrClusterNotFound):
		handler.WriteError(w, http.StatusNotFound, "集群未登记")
	default:
		clusterRegistryLog.Error(action, "err", err)
		handler.WriteError(w, http.StatusInternalServerError, action)
	}
}

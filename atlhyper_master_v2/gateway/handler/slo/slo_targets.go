// atlhyper_master_v2/gateway/handler/slo_targets.go
// SLO 目标管理与状态历史 Handler 方法
package slo

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"

	"AtlHyper/atlhyper_master_v2/gateway/handler"
	"AtlHyper/atlhyper_master_v2/model"
)

// ==================== 目标管理 ====================

// Targets 处理 /api/v2/slo/targets
func (h *SLOHandler) Targets(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.getTargets(w, r)
	case http.MethodPut:
		h.updateTarget(w, r)
	default:
		handler.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *SLOHandler) getTargets(w http.ResponseWriter, r *http.Request) {
	clusterID := r.URL.Query().Get("cluster_id")
	if clusterID == "" {
		clusterID = h.defaultClusterID(r.Context())
	}

	targets, err := h.querySvc.GetSLOTargets(r.Context(), clusterID)
	if err != nil {
		sloLog.Error("获取 targets 失败", "err", err)
		handler.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	handler.WriteJSON(w, http.StatusOK, targets)
}

func (h *SLOHandler) updateTarget(w http.ResponseWriter, r *http.Request) {
	var req model.UpdateSLOTargetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := validateTargetRequest(&req); err != nil {
		handler.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	if req.ClusterID == "" {
		req.ClusterID = h.defaultClusterID(r.Context())
	}

	if err := h.opsSvc.UpsertSLOTarget(r.Context(), &req); err != nil {
		sloLog.Error("更新 target 失败", "err", err)
		handler.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	handler.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ==================== 状态历史 ====================

// StatusHistory GET /api/v2/slo/status-history
// 状态历史不再写入 SQLite，返回空数组
func (h *SLOHandler) StatusHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		handler.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	handler.WriteJSON(w, http.StatusOK, []model.SLOStatusHistoryItem{})
}

// ==================== 写入校验 ====================

// SLO 目标的取值范围，与前端目标弹窗（SLOTargetModal）的选项一致
const (
	maxHostLen          = 253 // DNS 名称上限
	maxClusterIDLen     = 128
	maxP95LatencyTarget = 600_000 // 10 分钟（毫秒）
)

// sloWindowOptions 允许的滚动窗口天数；0 = 用默认值
var sloWindowOptions = map[int]bool{0: true, 1: true, 3: true, 7: true}

// sloHostPattern 域名（可带一层通配前缀 "*."），不含端口与路径
var sloHostPattern = regexp.MustCompile(`^(\*\.)?([A-Za-z0-9]([A-Za-z0-9-]{0,61}[A-Za-z0-9])?\.)*[A-Za-z0-9]([A-Za-z0-9-]{0,61}[A-Za-z0-9])?$`)

// validateTargetRequest 校验 SLO 目标写入请求，返回面向用户的错误说明
func validateTargetRequest(req *model.UpdateSLOTargetRequest) error {
	switch {
	case req.Host == "":
		return errors.New("host required")
	case len(req.Host) > maxHostLen || !sloHostPattern.MatchString(req.Host):
		return errors.New("host 必须是域名")
	case len(req.ClusterID) > maxClusterIDLen:
		return errors.New("clusterId 过长")
	case !sloWindowOptions[req.WindowDays]:
		return errors.New("windowDays 只能是 1、3 或 7")
	case req.AvailabilityTarget <= 0 || req.AvailabilityTarget > 100:
		return errors.New("availabilityTarget 必须在 (0, 100] 之间")
	case req.P95LatencyTarget <= 0 || req.P95LatencyTarget > maxP95LatencyTarget:
		return fmt.Errorf("p95LatencyTarget 必须在 1–%d 毫秒之间", maxP95LatencyTarget)
	}
	return nil
}

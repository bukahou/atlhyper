// atlhyper_master_v2/agentsdk/slo.go
// SLO 指标接收 Handler
package agentsdk

import (
	"encoding/json"
	"net/http"

	"AtlHyper/model_v2"
)

// handleSLO 处理 SLO 指标推送
//
// POST /agent/slo
// Body: model_v2.SLOPushRequest (JSON)
func (s *Server) handleSLO(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 检查 SLO Processor 是否启用
	if s.sloProcessor == nil {
		http.Error(w, "SLO not enabled", http.StatusServiceUnavailable)
		return
	}

	// 解析请求
	var req model_v2.SLOPushRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 获取集群 ID
	clusterID := req.ClusterID
	if clusterID == "" {
		clusterID = r.Header.Get("X-Cluster-ID")
	}
	if clusterID == "" {
		http.Error(w, "Missing cluster ID", http.StatusBadRequest)
		return
	}

	// 调用 SLO Processor 处理指标
	if err := s.sloProcessor.ProcessIngressMetrics(r.Context(), clusterID, &req.Metrics); err != nil {
		log.Warn("处理 SLO 指标失败", "cluster", clusterID, "err", err)
		http.Error(w, "Failed to process metrics", http.StatusInternalServerError)
		return
	}

	// 处理 IngressRoute 映射
	if len(req.IngressRoutes) > 0 {
		if err := s.sloProcessor.ProcessIngressRoutes(r.Context(), clusterID, req.IngressRoutes); err != nil {
			log.Warn("处理 IngressRoute 映射失败", "cluster", clusterID, "err", err)
			// 不返回错误，继续处理
		}
	}

	log.Debug("SLO 指标已接收",
		"cluster", clusterID,
		"counters", len(req.Metrics.Counters),
		"histograms", len(req.Metrics.Histograms),
		"routes", len(req.IngressRoutes),
	)

	w.WriteHeader(http.StatusAccepted)
}

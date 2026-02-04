// atlhyper_master_v2/agentsdk/heartbeat.go
// 处理 Agent 心跳
// 数据处理通过 Processor 层
package agentsdk

import (
	"encoding/json"
	"net/http"
)

// 使用 server.go 中定义的 log 变量

// handleHeartbeat 处理心跳请求
// POST /agent/heartbeat
// Header: X-Cluster-ID
func (s *Server) handleHeartbeat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 从 Header 获取 ClusterID（与其他 API 保持一致）
	clusterID := r.Header.Get("X-Cluster-ID")
	if clusterID == "" {
		http.Error(w, "X-Cluster-ID header is required", http.StatusBadRequest)
		return
	}

	// 通过 Processor 处理心跳
	if err := s.processor.ProcessHeartbeat(clusterID); err != nil {
		log.Error("处理心跳失败", "err", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(HeartbeatResponse{Status: "ok"})
}

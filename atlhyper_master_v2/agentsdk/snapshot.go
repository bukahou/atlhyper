// atlhyper_master_v2/agentsdk/snapshot.go
// 处理 Agent 快照上报
// 直接解析 model_v2.ClusterSnapshot 格式
package agentsdk

import (
	"compress/gzip"
	"encoding/json"
	"io"
	"log"
	"net/http"

	"AtlHyper/model_v2"
)

// handleSnapshot 处理快照上报
// POST /agent/snapshot
func (s *Server) handleSnapshot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 解压 Gzip（如果有）
	var reader io.Reader = r.Body
	if r.Header.Get("Content-Encoding") == "gzip" {
		gr, err := gzip.NewReader(r.Body)
		if err != nil {
			log.Printf("[AgentSDK] 创建 gzip 解码器失败: %v", err)
			http.Error(w, "Invalid gzip", http.StatusBadRequest)
			return
		}
		defer gr.Close()
		reader = gr
	}

	// 直接解析为 model_v2.ClusterSnapshot
	var snapshot model_v2.ClusterSnapshot
	if err := json.NewDecoder(reader).Decode(&snapshot); err != nil {
		log.Printf("[AgentSDK] 解析快照失败: %v", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// 获取 cluster_id（优先使用 Header，其次使用 body 中的值）
	clusterID := r.Header.Get("X-Cluster-ID")
	if clusterID == "" {
		clusterID = snapshot.ClusterID
	}

	if clusterID == "" {
		http.Error(w, "cluster_id is required", http.StatusBadRequest)
		return
	}

	// 确保 snapshot 中的 ClusterID 与 Header 一致
	snapshot.ClusterID = clusterID

	// 通过 Processor 处理
	if err := s.processor.ProcessSnapshot(clusterID, &snapshot); err != nil {
		log.Printf("[AgentSDK] 处理器错误: %v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

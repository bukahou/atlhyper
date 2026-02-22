// atlhyper_master_v2/agentsdk/command.go
// 处理指令下发（长轮询）
package agentsdk

import (
	"encoding/json"
	"net/http"
)

// 使用 server.go 中定义的 log 变量

// handleCommands 处理指令轮询
// GET /agent/commands?cluster_id=xxx
func (s *Server) handleCommands(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	clusterID := r.URL.Query().Get("cluster_id")
	if clusterID == "" {
		http.Error(w, "cluster_id is required", http.StatusBadRequest)
		return
	}

	// topic: ops / ai，默认 ops
	topic := r.URL.Query().Get("topic")
	if topic == "" {
		topic = "ops"
	}

	// 长轮询等待指令
	cmd, err := s.bus.WaitCommand(r.Context(), clusterID, topic, s.timeout)
	if err != nil {
		// 客户端断开连接
		if r.Context().Err() != nil {
			return
		}
		log.Error("等待指令出错", "err", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	// 返回响应
	resp := CommandResponse{HasCommand: false}
	if cmd != nil {
		resp.HasCommand = true
		resp.Command = cmd
		log.Debug("指令已下发", "cmd", cmd.ID, "cluster", clusterID, "action", cmd.Action, "source", cmd.Source)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

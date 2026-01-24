// atlhyper_master_v2/agentsdk/command.go
// 处理指令下发（长轮询）
package agentsdk

import (
	"encoding/json"
	"log"
	"net/http"
)

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

	// 长轮询等待指令
	cmd, err := s.bus.WaitCommand(r.Context(), clusterID, s.timeout)
	if err != nil {
		// 客户端断开连接
		if r.Context().Err() != nil {
			return
		}
		log.Printf("[AgentSDK] 等待指令出错: %v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	// 返回响应
	resp := CommandResponse{HasCommand: false}
	if cmd != nil {
		resp.HasCommand = true
		resp.Command = &CommandInfo{
			ID:              cmd.ID,
			Action:          cmd.Action,
			TargetKind:      cmd.TargetKind,
			TargetNamespace: cmd.TargetNamespace,
			TargetName:      cmd.TargetName,
			Params:          cmd.Params,
		}
		log.Printf("[AgentSDK] 指令已下发: id=%s, 集群=%s, 操作=%s",
			cmd.ID, clusterID, cmd.Action)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

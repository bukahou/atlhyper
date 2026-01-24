// atlhyper_master_v2/agentsdk/result.go
// 处理指令执行结果
package agentsdk

import (
	"encoding/json"
	"log"
	"net/http"

	"AtlHyper/atlhyper_master_v2/model"
)

// handleResult 处理执行结果上报
// POST /agent/result
func (s *Server) handleResult(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ResultRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[AgentSDK] 解析执行结果失败: %v", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.CommandID == "" {
		http.Error(w, "command_id is required", http.StatusBadRequest)
		return
	}

	// 转换为 Model 格式
	result := &model.CommandResult{
		Success: req.Success,
		Output:  req.Output,
		Error:   req.Error,
	}

	// 确认指令完成
	if err := s.bus.AckCommand(req.CommandID, result); err != nil {
		log.Printf("[AgentSDK] 确认指令完成失败: %v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	log.Printf("[AgentSDK] 已收到执行结果: 指令=%s, 成功=%v", req.CommandID, req.Success)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(ResultResponse{Status: "ok"})
}

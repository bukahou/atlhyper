// atlhyper_master_v2/agentsdk/result.go
// 处理指令执行结果
package agentsdk

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"AtlHyper/atlhyper_master_v2/model"
)

// 使用 server.go 中定义的 log 变量

// handleResult 处理执行结果上报
// POST /agent/result
func (s *Server) handleResult(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ResultRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Error("解析执行结果失败", "err", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.CommandID == "" {
		http.Error(w, "command_id is required", http.StatusBadRequest)
		return
	}

	// 转换为 Model 格式
	result := &model.CommandResult{
		CommandID: req.CommandID,
		Success:   req.Success,
		Output:    req.Output,
		Error:     req.Error,
	}

	// 确认指令完成
	if err := s.bus.AckCommand(req.CommandID, result); err != nil {
		log.Error("确认指令完成失败", "err", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	// 持久化执行结果到数据库
	s.persistResult(req.CommandID, result)

	log.Debug("已收到执行结果", "cmd", req.CommandID, "success", req.Success)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(ResultResponse{Status: "ok"})
}

// persistResult 持久化指令执行结果
func (s *Server) persistResult(cmdID string, result *model.CommandResult) {
	if s.cmdRepo == nil {
		return
	}

	ctx := context.Background()

	// 获取现有记录
	history, err := s.cmdRepo.GetByCommandID(ctx, cmdID)
	if err != nil || history == nil {
		log.Warn("获取指令历史失败或不存在", "cmd", cmdID, "err", err)
		return
	}

	// 更新结果
	now := time.Now()
	history.FinishedAt = &now
	if result.Success {
		history.Status = model.CommandStatusSuccess
	} else {
		history.Status = model.CommandStatusFailed
		history.ErrorMessage = result.Error
	}

	resultJSON, _ := json.Marshal(result)
	history.Result = string(resultJSON)

	if history.StartedAt == nil {
		history.StartedAt = &now
	}
	history.DurationMs = now.Sub(history.CreatedAt).Milliseconds()

	if err := s.cmdRepo.Update(ctx, history); err != nil {
		log.Error("更新指令历史失败", "cmd", cmdID, "err", err)
	}
}

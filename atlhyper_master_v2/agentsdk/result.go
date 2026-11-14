// atlhyper_master_v2/agentsdk/result.go
// 处理指令执行结果
package agentsdk

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"AtlHyper/model_v3/command"
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
	result := &command.Result{
		CommandID: req.CommandID,
		Success:   req.Success,
		Output:    req.Output,
		Error:     req.Error,
	}

	// 确认指令完成。回报方的集群交给队列核对：指令不属于它时不写入结果
	reporter := clusterFor(r, r.Header.Get("X-Cluster-ID"))
	if err := s.bus.AckCommand(reporter, req.CommandID, result); err != nil {
		if errors.Is(err, command.ErrNotOwnedByCluster) {
			log.Warn("拒绝回报不属于自己集群的指令结果", "cmd", req.CommandID, "reporter", reporter)
			http.Error(w, "command does not belong to this cluster", http.StatusForbidden)
			return
		}
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
func (s *Server) persistResult(cmdID string, result *command.Result) {
	if s.cmdRepo == nil {
		return
	}

	ctx := context.Background()

	// 获取现有记录
	history, err := s.cmdRepo.GetByCommandID(ctx, cmdID)
	if err != nil {
		log.Error("读取指令历史失败，执行结果未记入历史", "cmd", cmdID, "err", err)
		return
	}
	if history == nil {
		// 不经指令服务下发的指令（如部署器）没有历史记录，无需更新
		log.Debug("指令没有历史记录，跳过", "cmd", cmdID)
		return
	}

	// 更新结果
	now := time.Now()
	history.FinishedAt = &now
	if result.Success {
		history.Status = command.StatusSuccess
	} else {
		history.Status = command.StatusFailed
		history.ErrorMessage = result.Error
	}

	// 只存执行结论：Output 只交给同步等待方，不进入持久化
	resultJSON, _ := json.Marshal(result.Metadata())
	history.Result = string(resultJSON)

	if history.StartedAt == nil {
		history.StartedAt = &now
	}
	history.DurationMs = now.Sub(history.CreatedAt).Milliseconds()

	if err := s.cmdRepo.Update(ctx, history); err != nil {
		log.Error("更新指令历史失败", "cmd", cmdID, "err", err)
	}
}

// persistExpired 把未被执行的过期指令在历史里标成 timeout
func (s *Server) persistExpired(cmdID, reason string) {
	if s.cmdRepo == nil {
		return
	}
	ctx := context.Background()
	history, err := s.cmdRepo.GetByCommandID(ctx, cmdID)
	if err != nil || history == nil {
		return // 没有历史记录的指令（读查询）无需更新
	}
	now := time.Now()
	history.Status = command.StatusTimeout
	history.ErrorMessage = reason
	history.FinishedAt = &now
	history.DurationMs = now.Sub(history.CreatedAt).Milliseconds()
	if err := s.cmdRepo.Update(ctx, history); err != nil {
		log.Error("更新指令历史失败", "cmd", cmdID, "err", err)
	}
}

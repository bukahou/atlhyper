// atlhyper_master_v2/agentsdk/command.go
// 处理指令下发（长轮询）
package agentsdk

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"AtlHyper/model_v3/command"
)

// defaultCommandTTL 指令从创建到被 Agent 取走的最长等待时间。
//
// Agent 在线时指令几乎立刻被取走；超过这个时间还没取走，说明 Agent 当时不在线。
// 这样的指令不再下发 —— 否则几小时前点的「重启」会在 Agent 恢复连接的那一刻突然执行。
const defaultCommandTTL = 2 * time.Minute

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

	// 长轮询等待指令（过期的指令在这里被丢弃，不下发）
	cmd, err := s.waitLiveCommand(r.Context(), clusterID, topic)
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

// waitLiveCommand 长轮询取下一条仍有效的指令；途中遇到过期指令则判失败并继续取。
// 整个调用的等待时间不超过 s.timeout（Agent 的 HTTP 超时按它设定）。
func (s *Server) waitLiveCommand(ctx context.Context, clusterID, topic string) (*command.Command, error) {
	deadline := time.Now().Add(s.timeout)
	for {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return nil, nil
		}
		cmd, err := s.bus.WaitCommand(ctx, clusterID, topic, remaining)
		if err != nil || cmd == nil {
			return nil, err
		}
		if age := time.Since(cmd.CreatedAt); age > s.commandTTL() {
			s.expireCommand(cmd, age)
			continue
		}
		return cmd, nil
	}
}

func (s *Server) commandTTL() time.Duration {
	if s.cmdTTL > 0 {
		return s.cmdTTL
	}
	return defaultCommandTTL
}

// expireCommand 把过期指令判为失败：唤醒同步等待方，并把历史记录标成 timeout
func (s *Server) expireCommand(cmd *command.Command, age time.Duration) {
	reason := fmt.Sprintf("指令已过期：创建后 %s 才被 Agent 取到（上限 %s），未执行",
		age.Round(time.Second), s.commandTTL())
	log.Warn("丢弃过期指令", "cmd", cmd.ID, "cluster", cmd.ClusterID, "action", cmd.Action, "age", age.Round(time.Second))

	result := &command.Result{CommandID: cmd.ID, Success: false, Error: reason}
	if err := s.bus.AckCommand(cmd.ID, result); err != nil {
		log.Error("确认过期指令失败", "cmd", cmd.ID, "err", err)
	}
	s.persistExpired(cmd.ID, reason)
}

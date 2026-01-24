// atlhyper_master_v2/gateway/handler/command.go
// 指令下发 API Handler
// 创建指令通过 CommandService，查询状态通过 Query
package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"AtlHyper/atlhyper_master_v2/service"
	"AtlHyper/atlhyper_master_v2/service/operations"
)

// CommandHandler 指令 Handler
type CommandHandler struct {
	svc service.Service
}

// NewCommandHandler 创建 CommandHandler
func NewCommandHandler(svc service.Service) *CommandHandler {
	return &CommandHandler{svc: svc}
}

// CreateCommandRequest 创建指令请求
type CreateCommandRequest struct {
	ClusterID       string                 `json:"cluster_id"`
	Action          string                 `json:"action"`
	TargetKind      string                 `json:"target_kind,omitempty"`
	TargetNamespace string                 `json:"target_namespace,omitempty"`
	TargetName      string                 `json:"target_name,omitempty"`
	Params          map[string]interface{} `json:"params,omitempty"`
}

// Create 创建指令
func (h *CommandHandler) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req CreateCommandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// 创建指令
	resp, err := h.svc.CreateCommand(&operations.CreateCommandRequest{
		ClusterID:       req.ClusterID,
		Action:          req.Action,
		TargetKind:      req.TargetKind,
		TargetNamespace: req.TargetNamespace,
		TargetName:      req.TargetName,
		Params:          req.Params,
		Source:          "web",
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"command_id": resp.CommandID,
		"status":     resp.Status,
	})
}

// GetStatus 获取指令状态
func (h *CommandHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// 解析 commandID: /api/v2/commands/{id}
	path := strings.TrimPrefix(r.URL.Path, "/api/v2/commands/")
	commandID := strings.TrimSuffix(path, "/")
	if commandID == "" {
		writeError(w, http.StatusBadRequest, "command_id required")
		return
	}

	// 通过 Query 层查询状态
	status, err := h.svc.GetCommandStatus(r.Context(), commandID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get command status")
		return
	}
	if status == nil {
		writeError(w, http.StatusNotFound, "command not found")
		return
	}

	writeJSON(w, http.StatusOK, status)
}

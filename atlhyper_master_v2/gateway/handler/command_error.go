package handler

import (
	"context"
	"errors"
	"net/http"

	"AtlHyper/atlhyper_master_v2/model"
)

// WriteCommandError 把创建 / 等待指令的错误映射成状态码（按错误类别，⛔ 不比对文案）
func WriteCommandError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, model.ErrInvalidCommand):
		WriteError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, model.ErrClusterOffline):
		WriteError(w, http.StatusServiceUnavailable, "集群不在线：没有可接收指令的 Agent")
	case errors.Is(err, model.ErrEnqueueFailed):
		WriteError(w, http.StatusServiceUnavailable, "指令未能下发，请稍后重试")
	case errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
		WriteError(w, http.StatusGatewayTimeout, "等待 Agent 结果超时，请稍后重试")
	default:
		WriteError(w, http.StatusInternalServerError, "指令执行失败")
	}
}

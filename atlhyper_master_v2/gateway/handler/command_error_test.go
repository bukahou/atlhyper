package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/model_v3/command"
)

func TestWriteCommandError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want int
	}{
		{"参数不合法", fmt.Errorf("%w: replicas is required", model.ErrInvalidCommand), http.StatusBadRequest},
		{"来源不在名单", fmt.Errorf("%w: %w", model.ErrInvalidCommand, command.ErrActionNotPermitted), http.StatusBadRequest},
		{"集群不在线", fmt.Errorf("%w: c9", model.ErrClusterOffline), http.StatusServiceUnavailable},
		{"入队失败", fmt.Errorf("%w: full", model.ErrEnqueueFailed), http.StatusServiceUnavailable},
		{"等待超时", fmt.Errorf("wait command x: %w", context.DeadlineExceeded), http.StatusGatewayTimeout},
		{"其他错误", errors.New("boom"), http.StatusInternalServerError},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			WriteCommandError(rec, c.err)
			if rec.Code != c.want {
				t.Errorf("status = %d, want %d", rec.Code, c.want)
			}
		})
	}
}

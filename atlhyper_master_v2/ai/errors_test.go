package ai

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"AtlHyper/atlhyper_master_v2/database"
)

type emptyProviderRepo struct{ database.AIProviderRepository }

func (emptyProviderRepo) List(context.Context) ([]*database.AIProvider, error) { return nil, nil }

// B1：配置类错误要能被上层按类型识别（而不是靠匹配中文文案），网关据此给前端稳定的错误码
func TestLoadAIConfigForRole_UnassignedIsTyped(t *testing.T) {
	s := &aiServiceImpl{providerRepo: emptyProviderRepo{}}
	_, err := s.loadAIConfigForRole(context.Background(), RoleBackground)
	if !errors.Is(err, ErrRoleNotAssigned) {
		t.Errorf("err = %v，应可 errors.Is(ErrRoleNotAssigned)", err)
	}
	wrapped := errors.Join(errors.New("LLM 调用失败"), err)
	if !errors.Is(wrapped, ErrRoleNotAssigned) {
		t.Error("层层包装后仍应可识别")
	}
}

// C：错误码在 ai 包统一分类，网关响应与深度分析状态共用一份
func TestErrorCode(t *testing.T) {
	cases := []struct {
		err  error
		want string
	}{
		{fmt.Errorf("x: %w", ErrRoleNotAssigned), "ai_role_unassigned"},
		{fmt.Errorf("x: %w", ErrBudgetExhausted), "ai_budget_exhausted"},
		{fmt.Errorf("第 3 轮: %w", context.DeadlineExceeded), "ai_timeout"},
		{errors.New("AccessDeniedException"), "ai_upstream"},
		{nil, ""},
	}
	for _, c := range cases {
		if got := ErrorCode(c.err); got != c.want {
			t.Errorf("ErrorCode(%v) = %q, want %q", c.err, got, c.want)
		}
	}
}

package enricher

import (
	"errors"
	"fmt"
	"testing"

	"AtlHyper/atlhyper_master_v2/ai"
)

// AI 没配（角色未分配提供商）是配置状态不是故障：只提示一次；配好成功一次后，再没配会再提示一次；其余错误照常报错
func TestBackgroundFailureLevel(t *testing.T) {
	var notice unconfiguredNotice
	unassigned := fmt.Errorf("LLM 调用失败: AI 配置错误: %w", ai.ErrRoleNotAssigned)

	if got := notice.levelFor(unassigned); got != failureLevelWarnOnce {
		t.Errorf("第一次没配应提示一次，实际 %s", got)
	}
	if got := notice.levelFor(unassigned); got != failureLevelSilent {
		t.Errorf("之后不再重复，实际 %s", got)
	}
	if got := notice.levelFor(errors.New("超时")); got != failureLevelError {
		t.Errorf("其余错误照常报错，实际 %s", got)
	}
	notice.succeeded()
	if got := notice.levelFor(unassigned); got != failureLevelWarnOnce {
		t.Errorf("成功过之后又没配，应再提示一次，实际 %s", got)
	}
}

package prompts

import (
	"strings"
	"testing"
)

// A5：原因 / 根因必须有出处。实测模型曾按组件名常识编出「RBAC 权限」「OOMKilled」等重启原因，
// 而工具结果里根本没有这些字段 —— 提示词要把「原因从哪来、查不到怎么写」说具体
func TestChatPrompt_RequiresEvidenceForCauses(t *testing.T) {
	p := BuildChatPrompt()
	for _, must := range []string{
		"原因未知（证据不足）",      // 查不到时的固定写法
		"lastState",       // 重启原因的出处
		"不得根据组件名称或常识推测原因", // 明确禁止的行为
	} {
		if !strings.Contains(p, must) {
			t.Errorf("chat 提示词缺少：%s", must)
		}
	}
}

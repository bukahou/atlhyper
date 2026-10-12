package prompts

import (
	"strings"
	"testing"
)

// A7：证据规则须覆盖全部三个 AI 角色（A5 只覆盖了对话；后台分析实测仍按 Pod 名称推测原因，
// 并把 AIOps 状态变化写成「健康检查失败」）
func TestAllRolePrompts_ShareEvidenceRules(t *testing.T) {
	prompts := map[string]string{
		"chat":       BuildChatPrompt(),
		"analysis":   BuildAnalysisPrompt(),
		"background": BuildBackgroundPrompt(&IncidentPromptContext{}).System,
	}
	for role, p := range prompts {
		for _, must := range []string{
			"不得根据组件名称或常识推测原因",
			"原因未知（证据不足）",
			"AIOps 状态变化（healthy / warning / incident）不等于健康检查或探针失败",
			"没有证据时不得断言影响面",
		} {
			if !strings.Contains(p, must) {
				t.Errorf("%s 提示词缺少：%s", role, must)
			}
		}
	}
}

// A10：Pod 容器状态作为证据段落进入后台分析的用户消息（为空时不出现）
func TestBackgroundPrompt_IncludesPodStates(t *testing.T) {
	with := BuildBackgroundPrompt(&IncidentPromptContext{PodStates: "ns/pod/a:\n  容器 app: 当前 waiting(CrashLoopBackOff) | 重启 5 次 | 上次退出: Error\n"})
	if !strings.Contains(with.User, "## Pod 容器状态") || !strings.Contains(with.User, "上次退出: Error") {
		t.Errorf("应包含 Pod 容器状态段落：\n%s", with.User)
	}
	without := BuildBackgroundPrompt(&IncidentPromptContext{})
	if strings.Contains(without.User, "## Pod 容器状态") {
		t.Error("无数据时不应出现该段落")
	}
}

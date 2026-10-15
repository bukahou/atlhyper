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

// 深度分析与后台分析看到同一份证据（深度分析曾只有概要 / 实体 / 时间线）
func TestAnalysisUserPrompt_SharesEvidence(t *testing.T) {
	ctx := &IncidentPromptContext{
		IncidentSummary:   "事件概要: x",
		HistoricalContext: "历史相似事件: y",
		RecentErrorLogs:   "ERROR z",
		PodStates:         "ns/pod/a:\n  容器 app: 当前 running\n",
	}
	got := BuildAnalysisUserPrompt(ctx)
	for _, want := range []string{"事件概要: x", "历史相似事件: y", "## 最近 ERROR 日志", "## Pod 容器状态", "请开始调查"} {
		if !strings.Contains(got, want) {
			t.Errorf("深度分析 user prompt 缺 %q：\n%s", want, got)
		}
	}
}

// 资源名只能照抄上下文（曾把 ReplicaSet 名当成 Deployment 名）
func TestSystemPrompts_ExactResourceNames(t *testing.T) {
	for name, sys := range map[string]string{
		"background": BuildBackgroundPrompt(&IncidentPromptContext{}).System,
		"analysis":   BuildAnalysisPrompt(),
	} {
		if !strings.Contains(sys, "资源名称只能使用上下文中出现的完整名称") {
			t.Errorf("%s 系统提示词缺资源名规则", name)
		}
	}
}

// 持久化 event 与同时段事件作为证据段落出现（为空时不出现）
func TestPrompts_RenderEventsAndConcurrent(t *testing.T) {
	ctx := &IncidentPromptContext{K8sEvents: "- [Pod ns/p] Unhealthy（累计 6 次）", ConcurrentIncidents: "同时段其他事件 (1 个)"}
	for name, got := range map[string]string{
		"background": BuildBackgroundPrompt(ctx).User,
		"analysis":   BuildAnalysisUserPrompt(ctx),
	} {
		for _, want := range []string{"## K8s 事件（atlhyper 持久化", "Unhealthy（累计 6 次）", "## 同时段其他事件", "同时段其他事件 (1 个)"} {
			if !strings.Contains(got, want) {
				t.Errorf("%s 缺 %q", name, want)
			}
		}
	}
	if empty := BuildBackgroundPrompt(&IncidentPromptContext{}).User; strings.Contains(empty, "## K8s 事件") || strings.Contains(empty, "## 同时段") {
		t.Error("无数据时不应出现这两节")
	}
}

// 规则：不再说「事件只保留 1 小时」了事；同时段多工作负载出事要考虑共同依赖且须有证据
func TestEvidenceRules_PersistedEventsAndCommonDependency(t *testing.T) {
	for _, want := range []string{"get_event_history", "K8s 事件（atlhyper 持久化）", "共同依赖", "待验证假设"} {
		if !strings.Contains(EvidenceRules, want) {
			t.Errorf("EvidenceRules 缺 %q", want)
		}
	}
}

func TestToolDefinitions_IncludeEventHistory(t *testing.T) {
	for _, d := range GetToolDefinitions() {
		if d.Name == "get_event_history" {
			return
		}
	}
	t.Error("工具列表缺 get_event_history")
}

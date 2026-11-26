package mcpserver

import (
	"slices"
	"testing"

	"AtlHyper/atlhyper_master_v2/ai/toolkit"
	aitools "AtlHyper/atlhyper_master_v2/ai/tools"
)

// 真实的开放名单对着真实的工具目录核对：每个都已注册、都是只读、参数里没有 cluster_id
func TestExposedTools_ResolveAgainstRealCatalog(t *testing.T) {
	registry := toolkit.NewRegistry()
	aitools.Register(registry, aitools.Deps{}) // 只取定义，处理函数不会被调用
	tools, err := resolveExposedTools(registry, exposedTools)
	if err != nil {
		t.Fatal(err)
	}
	if len(tools) != len(exposedTools) {
		t.Fatalf("解析出 %d 个, want %d", len(tools), len(exposedTools))
	}
}

// 会调大模型并写报告的 analyze_incident 不是只读：不在名单里，硬加进去启动也会失败
func TestExposedTools_AnalyzeIncidentStaysOut(t *testing.T) {
	if slices.Contains(exposedTools, "analyze_incident") {
		t.Fatal("analyze_incident 不应在开放名单里")
	}
	registry := toolkit.NewRegistry()
	aitools.Register(registry, aitools.Deps{})
	if _, err := resolveExposedTools(registry, []string{"analyze_incident"}); err == nil {
		t.Error("analyze_incident 不是只读，开放它应失败")
	}
}

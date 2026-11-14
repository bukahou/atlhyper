// atlhyper_master_v2/ai/tool_policy.go
// AI 工具的权限：按「谁会看到工具结果」授权
//
// 对话里工具结果会出现在回复中，读者是调用者本人；深度分析的读者是能看 AI 报告的人。
// 每个工具读取的数据所需权限，与网关上读同一份数据的接口一致 —— 不能经 AI 拿到网关上拿不到的东西。
// 权限名沿用网关权限表（gateway/middleware/permission.go）的字符串，是两边的契约；
// ai 包不引用网关包，由调用方把判断函数传进来（gateway/routes_test 校验这些名字都在权限表里）。
package ai

import (
	"fmt"

	"AtlHyper/atlhyper_master_v2/ai/llm"
	"AtlHyper/atlhyper_master_v2/ai/prompts"
)

// PermissionCheck 判断会看到工具结果的人是否拥有某项权限
type PermissionCheck func(permission string) bool

// 工具读取的数据所需权限
const (
	permNone            = ""                    // 不需要额外权限（能进对话 / 分析即可）
	permWorkloadEnvRead = "workload.env.read"   // 原始资源对象（含 env 字面值）
	permEventsRead      = "events.read"         // K8s 事件
	permSecretRead      = "secret.read"         // ConfigMap 内容（网关上与 Secret 同档）
	permPodLogsRead     = "pod.logs.read"       // 容器日志
	permAIOpsRead       = "aiops.read"          // 风险、事件、实体详情
	permAIAnalyze       = "ai.analyze"          // 触发事件 AI 总结
	permTracesRead      = "traces.read"         // 链路
	permLogsRead        = "logs.read"           // OTel 日志正文
	permSLORead         = "slo.read"            // SLO
	permDeployHistory   = "deploy.history.read" // 部署历史
)

// toolPermissions 除 query_cluster（按动作细分，见 queryClusterPermission）外，每个工具所需的权限。
// ⛔ 新增工具必须在这里登记；没登记的工具一律拒绝。
var toolPermissions = map[string]string{
	"analyze_incident":      permAIAnalyze,
	"get_cluster_risk":      permAIOpsRead,
	"get_recent_incidents":  permAIOpsRead,
	"get_entity_detail":     permAIOpsRead,
	"get_event_history":     permEventsRead,
	"query_traces":          permTracesRead,
	"query_logs":            permLogsRead,
	"query_slo":             permSLORead,
	"get_deploy_history":    permDeployHistory,
	"rollback_deployment":   permDeployHistory, // 只读部署历史、生成回滚计划，不执行
	"github_read_file":      permNone,          // 能读哪些仓库由 GitHub 工具自身限定
	"github_search_code":    permNone,
	"github_recent_commits": permNone,
}

// ToolPermissionNames 工具用到的全部权限名（供网关测试核对它们都在权限表里）
func ToolPermissionNames() []string {
	return []string{
		permWorkloadEnvRead, permEventsRead, permSecretRead, permPodLogsRead, permAIOpsRead,
		permAIAnalyze, permTracesRead, permLogsRead, permSLORead, permDeployHistory,
	}
}

// queryClusterPermission query_cluster 按动作与资源类型细分
func queryClusterPermission(params map[string]interface{}) string {
	switch getString(params, "action") {
	case "get_logs":
		return permPodLogsRead
	case "get_configmap":
		return permSecretRead
	case "get_events":
		return permEventsRead
	default: // get / list / describe：返回原始对象；ConfigMap 的对象就是它的内容
		if getString(params, "kind") == "ConfigMap" {
			return permSecretRead
		}
		return permWorkloadEnvRead
	}
}

// toolPermission 返回调用该工具所需的权限；ok=false 表示未登记的工具
func toolPermission(tool string, params map[string]interface{}) (string, bool) {
	if tool == "query_cluster" {
		return queryClusterPermission(params), true
	}
	p, ok := toolPermissions[tool]
	return p, ok
}

// permits 判断能否调用某个工具；allow 为 nil 时一律拒绝
func permits(allow PermissionCheck, tool string, params map[string]interface{}) error {
	perm, ok := toolPermission(tool, params)
	if !ok {
		return fmt.Errorf("工具 %s 未登记权限，拒绝调用", tool)
	}
	if allow == nil {
		return fmt.Errorf("权限不足：未提供调用者权限，拒绝调用 %s", tool)
	}
	if perm != permNone && !allow(perm) {
		return fmt.Errorf("权限不足：调用 %s 需要 %s", tool, perm)
	}
	return nil
}

// toolDefsFor 只保留调用者能用的工具定义：发给模型的清单里不出现它调不了的工具。
// query_cluster 只要能读原始资源就保留，具体动作在执行时再按动作校验。
func toolDefsFor(allow PermissionCheck) []llm.ToolDefinition {
	if allow == nil {
		return nil
	}
	all := prompts.GetToolDefinitions()
	defs := make([]llm.ToolDefinition, 0, len(all))
	for _, d := range all {
		perm, ok := toolPermission(d.Name, map[string]interface{}{})
		if ok && (perm == permNone || allow(perm)) {
			defs = append(defs, d)
		}
	}
	return defs
}

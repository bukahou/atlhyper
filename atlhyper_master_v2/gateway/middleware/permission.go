// atlhyper_master_v2/gateway/middleware/permission.go
// 权限表：角色等级 → 权限集合（唯一权限源）
//
// 分档判据（设计文档 docs/design/active/permission-model-design.md）：
//   - 0 访客 = 成员：只含集群状态与形状
//   - 1 Viewer：内容级数据（可能带业务数据 / 个人信息）
//   - 2 Operator：凭据与写操作
//   - 3 Admin：系统配置
package middleware

// Permission 接口所需的能力，形如 "<领域>.<动作>"
type Permission string

const (
	// ---------- 0 级：集群状态与形状 ----------
	PermClusterRead    Permission = "cluster.read"     // 概览、资源列表 / 详情（env 字面值裁剪）
	PermEventsRead     Permission = "events.read"      // 事件
	PermMetricsRead    Permission = "metrics.read"     // 节点指标、硬件
	PermSLORead        Permission = "slo.read"         // SLO
	PermAPMSummaryRead Permission = "apm.summary.read" // APM 聚合：服务、统计、拓扑、操作
	PermLogsStatsRead  Permission = "logs.stats.read"  // 日志计数：概要、直方图
	PermAIOpsRead      Permission = "aiops.read"       // 依赖图、基线、风险、事件

	// ---------- 1 级：内容级数据 ----------
	PermWorkloadEnvRead   Permission = "workload.env.read"   // 资源详情中的 env 字面值
	PermLogsRead          Permission = "logs.read"           // 日志正文
	PermTracesRead        Permission = "traces.read"         // 链路列表 / 详情 / 属性相关性
	PermCommandsRead      Permission = "commands.read"       // 指令历史与状态
	PermDeployHistoryRead Permission = "deploy.history.read" // 部署历史
	PermGitHubStatusRead  Permission = "github.status.read"  // GitHub 连接状态
	PermAIChat            Permission = "ai.chat"             // AI 对话（有 LLM 成本）

	// ---------- 2 级：凭据与写操作 ----------
	PermSecretRead     Permission = "secret.read"      // Secret 列表 / 内容、ConfigMap 详情 / 内容
	PermPodLogsRead    Permission = "pod.logs.read"    // Pod 容器日志
	PermClusterOperate Permission = "cluster.operate"  // 指令下发、重启、扩缩容、改镜像、cordon
	PermAuditRead      Permission = "audit.read"       // 审计日志
	PermNotifyManage   Permission = "notify.manage"    // 通知渠道（读 + 写）
	PermAIAnalyze      Permission = "ai.analyze"       // AIOps AI 总结 / 建议 / 报告 / 深度分析
	PermAISettingsRead Permission = "ai.settings.read" // AI 配置查看

	// ---------- 3 级：系统配置 ----------
	PermUserManage      Permission = "user.manage"       // 用户管理（唯一的提权入口）
	PermAISettingsWrite Permission = "ai.settings.write" // AI 配置写入
	PermGitHubManage    Permission = "github.manage"     // GitHub 连接与仓库
	PermDeployManage    Permission = "deploy.manage"     // 部署配置与操作
)

// levelGrants 每一级新增的权限（高级自动继承低级，见 PermissionsFor）
var levelGrants = map[int][]Permission{
	RoleMember: {
		PermClusterRead, PermEventsRead, PermMetricsRead, PermSLORead,
		PermAPMSummaryRead, PermLogsStatsRead, PermAIOpsRead,
	},
	RoleViewer: {
		PermWorkloadEnvRead, PermLogsRead, PermTracesRead, PermCommandsRead,
		PermDeployHistoryRead, PermGitHubStatusRead, PermAIChat,
	},
	RoleOperator: {
		PermSecretRead, PermPodLogsRead, PermClusterOperate, PermAuditRead,
		PermNotifyManage, PermAIAnalyze, PermAISettingsRead,
	},
	RoleAdmin: {
		PermUserManage, PermAISettingsWrite, PermGitHubManage, PermDeployManage,
	},
}

// PermissionsFor 返回某一级拥有的全部权限（越界 level 返回空）
func PermissionsFor(level int) []Permission {
	if level < RoleMember || level > RoleAdmin {
		return []Permission{}
	}
	perms := []Permission{}
	for l := RoleMember; l <= level; l++ {
		perms = append(perms, levelGrants[l]...)
	}
	return perms
}

// Allowed 判断某一级是否拥有权限
func Allowed(level int, perm Permission) bool {
	for _, p := range PermissionsFor(level) {
		if p == perm {
			return true
		}
	}
	return false
}

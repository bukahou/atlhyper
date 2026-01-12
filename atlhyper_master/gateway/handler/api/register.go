// gateway/handler/api/register.go
// UI API 路由总注册
package api

import (
	"AtlHyper/atlhyper_master/gateway/handler/api/cluster"
	"AtlHyper/atlhyper_master/gateway/handler/api/event"
	"AtlHyper/atlhyper_master/gateway/handler/api/metrics"
	"AtlHyper/atlhyper_master/gateway/handler/api/ops"
	"AtlHyper/atlhyper_master/gateway/handler/api/overview"
	"AtlHyper/atlhyper_master/gateway/handler/api/system"
	"AtlHyper/atlhyper_master/gateway/handler/api/user"
	"AtlHyper/atlhyper_master/gateway/middleware/auth"

	"github.com/gin-gonic/gin"
)

// RegisterUIAPIRoutes 注册所有 UI API 路由
// =============================================================================
// 权限层级（从低到高）：
//
//   公开（OptionalAuth）    - 无 Token 也可访问，有 Token 则解析用户信息
//   Viewer（Role=1）       - 需要登录，最低权限
//   Operator（Role=2）     - 可执行操作，可查看敏感信息
//   Admin（Role=3）        - 最高权限，可管理用户和系统配置
//
// =============================================================================
// 路由结构与权限：
//
//   /uiapi
//   ├── /overview/*           总览页（公开）
//   ├── /cluster/*            集群资源页
//   │   ├── pod|node|...      公开查看
//   │   └── configmap         Operator（敏感信息）
//   ├── /event/*              事件日志页（公开）
//   ├── /metrics/*            监控指标页（公开）
//   ├── /ops/*                操作接口（Operator）
//   │   ├── pod/logs          查看日志（敏感）
//   │   └── pod/restart|...   执行操作
//   ├── /system/*             系统设置页
//   │   ├── /notify/get       通知配置查看（公开，低权限脱敏）
//   │   ├── /notify/update    通知配置修改（Admin）
//   │   └── /audit/*          审计日志（Viewer+）
//   └── /user/*               用户管理页
//       ├── /login            登录（公开）
//       ├── /list             用户列表（Viewer+）
//       └── /register|delete  用户管理（Admin）
//
// =============================================================================
func RegisterUIAPIRoutes(router *gin.RouterGroup) {
	// 全局可选认证：解析 Token（如有），后续可通过 c.Get("user_id") 判断是否登录
	router.Use(auth.OptionalAuth())

	// ==================== 公开查看模块 ====================

	// Overview - 总览页（公开）
	overview.Register(router)

	// Cluster - 集群资源页（公开查看，敏感信息需 Operator）
	cluster.Register(router)

	// Event - 事件日志页（公开）
	event.Register(router)

	// Metrics - 监控指标页（公开）
	metrics.Register(router)

	// ==================== 操作模块（Operator 权限） ====================

	// Ops - 操作接口（Pod 日志/重启, Node 封锁, Workload 扩缩容等）
	ops.Register(router)

	// ==================== 系统管理模块 ====================

	// System - 系统设置（通知配置 + 审计日志）
	system.Register(router)

	// User - 用户管理（登录公开，列表需登录，管理需 Admin）
	user.Register(router)
}

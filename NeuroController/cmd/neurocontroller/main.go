package main

import (
	"NeuroController/config"
	"NeuroController/db/sqlite"
	"NeuroController/external"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func main() {
	config.LoadConfig()

	// ✅ 设置结构化日志
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	// ✅ 初始化 SQLite 数据库
	sqlite.InitDB() 

	// ✅ 启动外部系统（邮件、Slack、Webhook 等）
	external.StartExternalSystems()
}


	// ✅ 初始化 controller-runtime client（含 rest.Config）
	// utils.InitK8sClient()

	// ✅ 初始化 metrics.k8s.io 客户端（需要在 InitK8sClient 之后）
	// utils.InitMetricsClient()

	// ✅ 启动内部子系统（诊断器、清理器等）
	// internal.StartInternalSystems()

	// ✅ 启动 controller-runtime 控制器管理器
	// bootstrap.StartManager()

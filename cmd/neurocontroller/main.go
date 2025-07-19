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

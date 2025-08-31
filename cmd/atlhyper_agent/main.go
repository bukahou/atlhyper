package main

import (
	agent "AtlHyper/atlhyper_agent"
	"AtlHyper/atlhyper_agent/bootstrap"
	"AtlHyper/config"

	"AtlHyper/atlhyper_agent/utils"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)


func main() {
	config.LoadConfig()

	// ✅ 设置结构化日志
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	// ✅ 初始化 controller-runtime client（含 rest.Config）
	utils.InitK8sClient()

	// ✅ 初始化 metrics.k8s.io 客户端（需要在 InitK8sClient 之后）
	utils.InitMetricsClient()

	// ✅ 启动内部子系统（诊断器、清理器等）
	agent.StartInternalSystems()

	// ✅ 启动事件推送（独立 goroutine，内部自行取 clusterID/定时/优雅退出）
	// go push.StartPusher() 

	// // ✅ 启动 Agent HTTP Server
	// go bootstrapgo.StartAgentServer()

	// ✅ 启动 controller-runtime 控制器管理器
	bootstrap.StartManager()
}

// =======================================================================================
// 📄 cmd/controller/main.go
//
// ✨ Description:
//     Entry point of NeuroController. This is a Kubernetes controller plugin designed
//     to run persistently inside the cluster. It dynamically enables modules such as
//     Watcher, Webhook, Scaler, Reporter, and NeuroAI based on the config.yaml file.
//
// 🧠 Startup Logic:
//     1. Initialize the logging system (zap)
//     2. Load configuration from config.yaml
//     3. Initialize Kubernetes client (controller-runtime)
//     4. Start modules in parallel as defined in the configuration
//     5. Enter the main event loop to monitor and respond to cluster events
//
// 📍 Deployment Recommendation:
//     - Deploy as a Kubernetes Deployment or DaemonSet
//     - Supports per-module enable/disable to fit different environments
//
// ✍️ Author: bukahou (@ZGMF-X10A)
// 📅 Created: June 2025
// =======================================================================================

package main

import (
	"NeuroController/config"
	"NeuroController/external"
	"NeuroController/internal"
	"NeuroController/internal/bootstrap"
	"NeuroController/internal/utils"

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
	internal.StartInternalSystems()

	// ✅ 启动外部系统（邮件、Slack、Webhook 等）
	external.StartExternalSystems()

	// ✅ 启动 controller-runtime 控制器管理器
	bootstrap.StartManager()
}

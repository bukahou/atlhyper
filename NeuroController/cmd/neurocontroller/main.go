// =======================================================================================
// 📄 cmd/neurocontroller/main.go
//
// 🧠 Entry Point of NeuroController
//
// 🔍 Overview:
//     NeuroController is a plugin-based Kubernetes controller that runs persistently
//     within the cluster. It initializes core components such as logging, configuration,
//     Kubernetes clients, diagnostics, and alerting systems.
//
// ⚙️ Startup Flow:
//     1. Initialize structured logging (Zap)
//     2. Load configuration (from environment or config map)
//     3. Initialize Kubernetes client (controller-runtime + rest.Config)
//     4. Initialize metrics.k8s.io client (optional)
//     5. Launch internal systems (e.g., diagnostics, cleaner)
//     6. Launch external systems (e.g., Email, Slack, Webhook)
//     7. Start controller manager (controller-runtime)
//
// 🚀 Deployment:
//     - Recommended to deploy as a Kubernetes Deployment (DaemonSet also supported)
//     - Modules can be enabled/disabled independently for flexibility
//     - Lightweight resource usage; ideal for Raspberry Pi or edge environments
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

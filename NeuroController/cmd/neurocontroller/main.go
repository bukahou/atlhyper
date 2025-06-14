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
	"NeuroController/internal/bootstrap"
	"NeuroController/internal/diagnosis"
	"NeuroController/internal/utils"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func main() {

	config.LoadConfig()
	// ✅ 设置 controller-runtime 的日志系统（应最先调用）
	ctrl.SetLogger(zap.New(zap.UseDevMode(false))) // (true): 开发模式 / (false): 生产模式
	utils.InitLogger()                             // 初始化 zap 日志记录器

	cfg := utils.InitK8sClient() // 初始化 Kubernetes 客户端配置
	// ✅ 自动选择最佳的 API Server 端点（集群内或集群外）
	// api := utils.ChooseBestK8sAPI(cfg.Host)
	utils.StartK8sHealthChecker(cfg) // 启动 K8s API 健康检查机制

	// ✅ 启动日志事件池的定时清理器（每 30 秒运行一次）
	diagnosis.StartDiagnosisSystem()

	// ✅ 注册模块并启动控制器管理器
	bootstrap.StartManager()
}

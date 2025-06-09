// =======================================================================================
// 📄 cmd/controller/main.go
//
// ✨ 功能说明：
//     NeuroController 的主启动入口，用作 Kubernetes 控制器插件的主服务，
//     长期运行于集群中，按配置文件动态启用 Watcher、Webhook、Scaler、Reporter、NeuroAI 等模块。
//
// 🧠 启动逻辑：
//     1. 初始化日志系统（zap）
//     2. 加载配置文件（config.yaml）
//     3. 初始化 Kubernetes 客户端（controller-runtime）
//     4. 根据配置按需启动各模块（可并发）
//     5. 持续运行监听并响应系统事件
//
// 📍 部署建议：
//     - 推荐部署为 Kubernetes 中的 Deployment 或 DaemonSet
//     - 支持模块启停配置，可根据不同环境动态裁剪功能
//
// ✍️ 作者：武夏锋（@ZGMF-X10A）
// 📅 创建时间：2025-06
// =======================================================================================

package main

import (
	"NeuroController/internal/bootstrap"
	"NeuroController/internal/diagnosis"
	"NeuroController/internal/utils"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func main() {
	// ✅ 设置 controller-runtime 日志系统（推荐放在最前）
	ctrl.SetLogger(zap.New(zap.UseDevMode(false))) //  (true)用于开发模式/(false)用于生产模式
	utils.InitLogger()

	cfg := utils.InitK8sClient()
	// ✅ 自动选择可用 API 地址（支持集群内外切换）
	// api := utils.ChooseBestK8sAPI(cfg.Host)
	utils.StartK8sHealthChecker(cfg)

	// ✅ 启动定时清理器（每 30 秒清理一次日志池）
	diagnosis.StartDiagnosisSystem()

	bootstrap.StartManager()
}

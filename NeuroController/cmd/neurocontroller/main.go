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

package neurocontroller

import (
	"NeuroController/internal/bootstrap"
	"NeuroController/internal/utils"
)

func main() {
	utils.InitLogger()
	utils.InitK8sClient()

	bootstrap.StartManager()
}

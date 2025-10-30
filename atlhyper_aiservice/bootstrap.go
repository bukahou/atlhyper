// atlhyper_aiservice/bootstrap.go
package bootstrap

import (
	"AtlHyper/atlhyper_aiservice/server"
	"log"
)

// StartAIService —— 启动 AI Service 核心组件
// ------------------------------------------------------------
// 负责启动整个 AI Service 服务，包括：
// 1️⃣ 初始化日志与环境
// 2️⃣ 启动 HTTP 服务
// 底层的 AI 客户端（Gemini / GPT 等）在运行时按需创建，无需预初始化。
func StartAIService() {
	log.Println("🧠 初始化 AI Service 系统组件 ...")

	// ✅ 启动 HTTP 服务（测试接口、AI 推理接口等）
	server.StartHTTPServer()

	log.Println("✅ AtlHyper AI Service 启动完成")
}

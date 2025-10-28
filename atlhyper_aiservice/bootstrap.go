// atlhyper_aiservice/bootstrap.go
package bootstrap

import (
	"AtlHyper/atlhyper_aiservice/client"
	"AtlHyper/atlhyper_aiservice/server"
	"log"
)

// StartAIService 启动 AI Service 的全部功能模块
func StartAIService() {
	log.Println("🧠 初始化 AI Service 系统组件 ...")

	// ✅ 初始化 Gemini 客户端（单例）
	client.InitGeminiClient()

	// ✅ 启动 HTTP 服务（测试接口、后续 AI 处理接口等）
	server.StartHTTPServer()

	log.Println("✅ AtlHyper AI Service 启动完成")
}

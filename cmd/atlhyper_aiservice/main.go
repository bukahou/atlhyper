// cmd/atlhyper_aiservice/main.go
package main

import (
	bootstrap "AtlHyper/atlhyper_aiservice"
	"AtlHyper/atlhyper_aiservice/config"
	"log"
)

func main() {
	log.Println("🚀 启动 AtlHyper AI Service")

	// ✅ 加载配置（从环境变量）
	config.MustLoad()

	// ✅ 启动服务（客户端 + HTTP）
	bootstrap.StartAIService()
}

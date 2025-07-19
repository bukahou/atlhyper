package bootstrapgo

import (
	"log"

	"NeuroController/sync/agent/server"

	"github.com/gin-gonic/gin"
)

// StartAgentServer 启动 Agent 的 HTTP Server（不写死任何路由前缀）
func StartAgentServer() {
	r := gin.Default()

	// 由外部决定前缀，这里直接挂载全部路由
	server.RegisterRoutes(r.Group("")) // 不加前缀，交由上层控制

	if err := r.Run(":8082"); err != nil {
		log.Fatalf("❌ Agent 启动失败: %v", err)
	}
}

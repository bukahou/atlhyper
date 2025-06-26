// 📄 external/webhook/dockerHub/register.go

package dockerHub

import (
	"github.com/gin-gonic/gin"
)

// RegisterDockerHubWebhook 注册 DockerHub webhook 路由（POST）
func RegisterDockerHubWebhook(rg *gin.RouterGroup) {
	rg.POST("/webhook/dockerhub", HandleDockerHubWebhook)
}

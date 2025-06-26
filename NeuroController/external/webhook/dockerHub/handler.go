// 📄 external/webhook/dockerHub/handler.go

package dockerHub

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

// HandleDockerHubWebhook 兼容 Gin 的 Webhook 入口
func HandleDockerHubWebhook(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.String(http.StatusBadRequest, "读取请求体失败: %v", err)
		return
	}
	defer c.Request.Body.Close()

	if err := ParseAndApplyDockerHubWebhook(body); err != nil {
		c.String(http.StatusInternalServerError, "Webhook 处理失败: %v", err)
		return
	}

	c.String(http.StatusOK, "✅ Deployment 已更新")
}

// external/audit/middleware_auto.go
package audit

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

var riskyPrefixes = []string{
	"/uiapi/pod-ops/",
	"/uiapi/deployment-ops/",
	"/uiapi/node-ops/",
	"/uiapi/auth/user/register",
	"/uiapi/auth/user/update-role",
}

// Auto 返回自动审计中间件
// - 失败兜底：所有 status >= 400 自动记失败
// - 高风险接口：成功也记（可按需开关 riskyOnSuccess）
func Auto(riskyOnSuccess bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		status := c.Writer.Status()
		_ = start // 目前没用到耗时，如需可加 latency 相关列后写

		// 失败兜底
		if status >= 400 {
			_ = Record(c, "auto."+safeFullPath(c), false, status)
			return
		}

		// 高风险成功也记
		if riskyOnSuccess && isRisky(c.Request.URL.Path) {
			_ = Record(c, "auto."+safeFullPath(c), true, status)
		}
	}
}

func isRisky(p string) bool {
	for _, pre := range riskyPrefixes {
		if strings.HasPrefix(p, pre) {
			return true
		}
	}
	return false
}

func safeFullPath(c *gin.Context) string {
	if c != nil && c.FullPath() != "" {
		return strings.TrimPrefix(c.FullPath(), "/")
	}
	if c != nil && c.Request != nil && c.Request.URL != nil {
		return strings.TrimPrefix(c.Request.URL.Path, "/")
	}
	return "unknown"
}

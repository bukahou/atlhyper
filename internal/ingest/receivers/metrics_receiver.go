// internal/ingest/receivers/metrics.go
package receivers

import (
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"NeuroController/internal/agent_store"
	nmodel "NeuroController/model/metrics"
)

const defaultMaxBodyBytes int64 = 2 << 20 // 2 MiB

// HandlePostMetrics 处理 collector 推送过来的 metrics 快照。
// 支持 Content-Encoding: gzip；做基本校验并写入全局 agent_store。
func HandlePostMetrics(c *gin.Context) {
	// 方法保护（通常不会触发，因为路由只绑定了 POST）
	if c.Request.Method != http.MethodPost {
		c.AbortWithStatus(http.StatusMethodNotAllowed)
		return
	}

	// 请求体大小限制（防止打爆内存）
	body := c.Request.Body
	if defaultMaxBodyBytes > 0 {
		body = http.MaxBytesReader(c.Writer, body, defaultMaxBodyBytes)
	}
	defer body.Close() // 注意：放在 MaxBytesReader 包装之后

	// 可选 gzip 解压
	var reader io.Reader = body
	if strings.EqualFold(c.GetHeader("Content-Encoding"), "gzip") {
		zr, err := gzip.NewReader(body)
		if err != nil {
			c.String(http.StatusBadRequest, "bad gzip body")
			return
		}
		defer zr.Close()
		reader = zr
	}

	// 反序列化为统一模型
	var snap nmodel.NodeMetricsSnapshot
	if err := json.NewDecoder(reader).Decode(&snap); err != nil {
		// 大小超限时，这里通常是 *http.MaxBytesError
		if _, ok := err.(*http.MaxBytesError); ok {
			c.AbortWithStatus(http.StatusRequestEntityTooLarge)
			return
		}
		c.String(http.StatusBadRequest, "invalid json")
		return
	}

	// 基本校验与兜底
	if snap.NodeName == "" {
		c.String(http.StatusBadRequest, "nodeName required")
		return
	}
	if snap.Timestamp.IsZero() {
		snap.Timestamp = time.Now()
	}

	// 入全局内存存储（只保留各节点“最新一条”）
	agent_store.PutSnapshot(&snap)

	// 202 表示已接收
	c.Status(http.StatusAccepted)
}

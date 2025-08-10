package receivers

import (
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"NeuroController/internal/ingest/store"
	nmodel "NeuroController/model/metrics"
)

// RegisterMetricsRoutes 注册 metrics 接收路由。
//   - r: 建议传入 /ingest 前缀下的分组，例如 r := engine.Group("/ingest")
//   - st: 内存存储（metrics_store）
//   - maxBodyBytes: 请求体大小限制（字节）；<=0 则使用默认 2MiB
func RegisterMetricsRoutes(r *gin.RouterGroup, st *store.Store, maxBodyBytes int64) {
	if maxBodyBytes <= 0 {
		maxBodyBytes = 2 << 20 // 2 MiB
	}
	// 选项 B：语义更明确
	r.POST("/v1/snapshot", func(c *gin.Context) {
		handlePostMetrics(c, st, maxBodyBytes)
	})

}

// handlePostMetrics 处理 collector 推送过来的 metrics 快照。
// 支持 Content-Encoding: gzip；做基本校验并写入内存 store。
func handlePostMetrics(c *gin.Context, st *store.Store, maxBody int64) {
	// 方法保护（通常不会触发，因为路由只绑定了 POST）
	if c.Request.Method != http.MethodPost {
		c.AbortWithStatus(http.StatusMethodNotAllowed)
		return
	}

	// 请求体大小限制（防止打爆内存）
	body := c.Request.Body
	defer body.Close()
	if maxBody > 0 {
		body = http.MaxBytesReader(c.Writer, body, maxBody)
	}

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

	// 反序列化为你们统一的模型结构
	var snap nmodel.NodeMetricsSnapshot
	if err := json.NewDecoder(reader).Decode(&snap); err != nil {
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

	// 入内存存储（10分钟窗口由 store 的清理协程负责）
	st.Put(snap.NodeName, &snap)

	// 202 表示已接收并入队/入库（此处直接入库）
	c.Status(http.StatusAccepted)
}

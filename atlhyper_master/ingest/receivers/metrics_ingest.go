package receivers

import (
	"encoding/json"
	"io"
	"net/http"

	"AtlHyper/atlhyper_master/master_store"
	"AtlHyper/model"
	"AtlHyper/model/envelope"
	ziputil "AtlHyper/utils" // 根目录通用 gzip 工具

	"github.com/gin-gonic/gin"
)

// 此端点只接收 metrics 快照
const srcMetricsSnapshot = model.SourceMetricsSnapshot

// HandleMetricsSnapshotIngest 处理 /ingest/metrics/snapshot
// - 兼容压缩与未压缩：支持 Content-Encoding:gzip，且自动嗅探魔数
// - 校验必要字段：Version / ClusterID / TimestampMs / Payload
// - 成功返回 204，无响应体
func HandleMetricsSnapshotIngest(c *gin.Context) {
	// 1) 限制“压缩前”的请求体大小（metrics 较大也通常 <2MiB 压缩后更小）
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 2<<20) // 2MiB

	// 2) 自动解压或透传（头部 + 魔数嗅探）
	rc, err := ziputil.MaybeGunzipReaderAuto(c.Request.Body, c.GetHeader("Content-Encoding"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "解压失败或请求体非法"})
		return
	}
	defer rc.Close()

	// （可选）3) 再限制“解压后”的最大可读字节，防 zip-bomb（按需调整）
	rcLimited := io.LimitReader(rc, 16<<20) // 16MiB

	// 4) 解析 JSON Envelope
	var env envelope.Envelope
	if err := json.NewDecoder(rcLimited).Decode(&env); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "请求体解析失败：不是有效的 JSON Envelope"})
		return
	}

	// 5) 基本校验
	if env.Source != srcMetricsSnapshot {
		c.JSON(http.StatusBadRequest, gin.H{"message": "source 非法：仅支持 metrics_snapshot"})
		return
	}
	if env.Version == "" || env.ClusterID == "" || env.TimestampMs <= 0 || len(env.Payload) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Envelope 字段缺失：version/cluster_id/ts_ms/payload 均为必填"})
		return
	}

	// 6) 入池（Master 全局内存池；后续由消费者解码处理）
	master_store.AppendEnvelope(env)

	// 7) 成功：204 无 body
	c.Status(http.StatusNoContent)
}

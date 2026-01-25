// gateway/handler/ingest/receivers/metrics_ingest.go
package receivers

import (
	"AtlHyper/atlhyper_master/service/ingest"

	"github.com/gin-gonic/gin"
)

// HandleMetricsSnapshotIngest 处理 /ingest/metrics/snapshot
// -----------------------------------------------------------------------------
// 数据流: Gateway → Service → Repository → DataHub
// 说明: 指标快照使用 ReplaceLatest 模式（替换最新）
// -----------------------------------------------------------------------------
func HandleMetricsSnapshotIngest(c *gin.Context) {
	// 1) 解析 Envelope（指标数据中等大小）
	env, ok := ParseEnvelope(c, MediumParseConfig)
	if !ok {
		return
	}

	// 2) 调用 Service 层处理
	if err := ingest.Default().ProcessMetricsSnapshot(c.Request.Context(), *env); err != nil {
		RespondValidationError(c, err)
		return
	}

	// 3) 成功响应
	RespondSuccess(c)
}

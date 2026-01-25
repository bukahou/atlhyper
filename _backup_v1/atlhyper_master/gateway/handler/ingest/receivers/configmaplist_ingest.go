// gateway/handler/ingest/receivers/configmaplist_ingest.go
package receivers

import (
	"AtlHyper/atlhyper_master/service/ingest"

	"github.com/gin-gonic/gin"
)

// HandleConfigMapListSnapshotIngest 处理 /ingest/configmaplist
// -----------------------------------------------------------------------------
// 数据流: Gateway → Service → Repository → DataHub
// -----------------------------------------------------------------------------
func HandleConfigMapListSnapshotIngest(c *gin.Context) {
	// 1) 解析 Envelope
	env, ok := ParseEnvelope(c, DefaultParseConfig)
	if !ok {
		return
	}

	// 2) 调用 Service 层处理
	if err := ingest.Default().ProcessConfigMapList(c.Request.Context(), *env); err != nil {
		RespondValidationError(c, err)
		return
	}

	// 3) 成功响应
	RespondSuccess(c)
}

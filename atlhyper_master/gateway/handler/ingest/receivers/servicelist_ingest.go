// gateway/handler/ingest/receivers/servicelist_ingest.go
package receivers

import (
	"AtlHyper/atlhyper_master/service/ingest"

	"github.com/gin-gonic/gin"
)

// HandleServiceListSnapshotIngest 处理 /ingest/servicelist
// -----------------------------------------------------------------------------
// 数据流: Gateway → Service → Repository → DataHub
// -----------------------------------------------------------------------------
func HandleServiceListSnapshotIngest(c *gin.Context) {
	// 1) 解析 Envelope
	env, ok := ParseEnvelope(c, DefaultParseConfig)
	if !ok {
		return
	}

	// 2) 调用 Service 层处理
	if err := ingest.Default().ProcessServiceList(c.Request.Context(), *env); err != nil {
		RespondValidationError(c, err)
		return
	}

	// 3) 成功响应
	RespondSuccess(c)
}

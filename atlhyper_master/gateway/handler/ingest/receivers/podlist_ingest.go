// gateway/handler/ingest/receivers/podlist_ingest.go
package receivers

import (
	"AtlHyper/atlhyper_master/service/ingest"

	"github.com/gin-gonic/gin"
)

// HandlePodListSnapshotIngest 处理 /ingest/podlist
// -----------------------------------------------------------------------------
// 数据流: Gateway → Service → Repository → DataHub
// -----------------------------------------------------------------------------
func HandlePodListSnapshotIngest(c *gin.Context) {
	// 1) 解析 Envelope
	env, ok := ParseEnvelope(c, DefaultParseConfig)
	if !ok {
		return
	}

	// 2) 调用 Service 层处理
	if err := ingest.Default().ProcessPodList(c.Request.Context(), *env); err != nil {
		RespondValidationError(c, err)
		return
	}

	// 3) 成功响应
	RespondSuccess(c)
}

// gateway/handler/ingest/receivers/events_receivers.go
package receivers

import (
	"AtlHyper/atlhyper_master/service/ingest"

	"github.com/gin-gonic/gin"
)

// HandleEventLogIngest 处理 /ingest/events/v1/eventlog
// -----------------------------------------------------------------------------
// 数据流: Gateway → Service → Repository → DataHub
// 说明: 事件使用 Append 模式（增量追加）
// -----------------------------------------------------------------------------
func HandleEventLogIngest(c *gin.Context) {
	// 1) 解析 Envelope（事件数据较小，使用小配置）
	env, ok := ParseEnvelope(c, SmallParseConfig)
	if !ok {
		return
	}

	// 2) 调用 Service 层处理
	if err := ingest.Default().ProcessEvents(c.Request.Context(), *env); err != nil {
		RespondValidationError(c, err)
		return
	}

	// 3) 成功响应
	RespondSuccess(c)
}

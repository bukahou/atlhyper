package dataapi

import (
	"NeuroController/internal/ingest/store"

	"github.com/gin-gonic/gin"
)

// r 是 /agent/dataapi
func RegisterRoutes(r *gin.RouterGroup, st *store.Store) {

	h := NewMetricsHandlers(st)
	// ✅ 测试/调试：全量导出
	r.GET("/all", h.GetAll)

}

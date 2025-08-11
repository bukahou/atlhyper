package dataapi

import (
	iface "NeuroController/interfaces/data_api" // 引入 interfaces 层
	"NeuroController/internal/ingest/store"

	"github.com/gin-gonic/gin"
)

// r 是 /agent/dataapi
func RegisterRoutes(r *gin.RouterGroup, st *store.Store) {
	// 用底层 store 构造 interfaces 层实例
	api := iface.NewMetricsStoreAPI(st)

	// 再用 interfaces 实例构造 handler
	h := NewMetricsHandlers(api)

	// ✅ 测试/调试：全量导出
	r.GET("/all", h.GetAll)

	// 最新数据
	r.GET("/latest", h.GetLatest)

	// 可选：时间区间查询
	r.GET("/range", h.GetRange)
}

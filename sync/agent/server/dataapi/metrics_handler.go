package dataapi

import (
	"net/http"

	"NeuroController/internal/ingest/store"

	"github.com/gin-gonic/gin"
)

// MetricsHandlers 持有依赖（这里是内存 store）
type MetricsHandlers struct {
	st *store.Store
}

// NewMetricsHandlers 构造函数，注入依赖
func NewMetricsHandlers(st *store.Store) *MetricsHandlers {
	return &MetricsHandlers{st: st}
}

// GetAll 返回当前 store 中保存的所有快照。
// GET /agent/dataapi/metrics/all
func (h *MetricsHandlers) GetAll(c *gin.Context) {
	data := h.st.DumpAll()
	c.JSON(http.StatusOK, data)
}

package dataapi

// import (
// 	"net/http"

// 	"NeuroController/internal/ingest/store"

// 	"github.com/gin-gonic/gin"
// )

// // GET /agent/dataapi/metrics/all
// // 直接返回 store 中当前保存的所有快照：map[nodeName][]NodeMetricsSnapshot
// func RegisterMetricsReadRoutes(r *gin.RouterGroup, st *store.Store) {
// 	r.GET("/all", func(c *gin.Context) {
// 		c.JSON(http.StatusOK, st.DumpAll())
// 	})
// }

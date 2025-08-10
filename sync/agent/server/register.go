package server

import (
	"NeuroController/internal/ingest/store"
	"NeuroController/sync/agent/server/commonapi"
	"NeuroController/sync/agent/server/dataapi"
	"NeuroController/sync/agent/server/uiapi"

	"github.com/gin-gonic/gin"
)

// sync/agent/server/register.go
func RegisterAllAgentRoutes(r *gin.RouterGroup, st *store.Store) {
	commonapi.RegisterCommonAPIRoutes(r.Group("/commonapi"))
	uiapi.RegisterUIRoutes(r.Group("/uiapi"))

	dataapi.RegisterRoutes(r.Group("/dataapi"), st)
	// dataapi.RegisterMetricsReadRoutes(dataGroup.Group("/metrics"), st)
}

// func RegisterAllAgentRoutes(r *gin.RouterGroup) {
// 	commonapi.RegisterCommonAPIRoutes(r.Group("/commonapi"))
// 	uiapi.RegisterUIRoutes(r.Group("/uiapi"))

// 	dataGroup := r.Group("/dataapi")
// 	dataapi.RegisterMetricsReadRoutes(dataGroup.Group("/metrics"), st)
// }
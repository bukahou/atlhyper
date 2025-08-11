package server

import (
	"NeuroController/internal/ingest/store"
	clusterapi "NeuroController/sync/agent/server/clusterapi"
	"NeuroController/sync/agent/server/commonapi"
	"NeuroController/sync/agent/server/dataapi"

	"github.com/gin-gonic/gin"
)

// sync/agent/server/register.go
func RegisterAllAgentRoutes(r *gin.RouterGroup, st *store.Store) {
	commonapi.RegisterCommonAPIRoutes(r.Group("/commonapi"))
	clusterapi.RegisterUIRoutes(r.Group("/uiapi"))

	dataapi.RegisterRoutes(r.Group("/dataapi"), st)
}

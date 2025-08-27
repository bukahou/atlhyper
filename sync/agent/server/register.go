package server

import (
	clusterapi "NeuroController/sync/agent/server/clusterapi"
	"NeuroController/sync/agent/server/commonapi"

	"github.com/gin-gonic/gin"
)

// sync/agent/server/register.go
func RegisterAllAgentRoutes(r *gin.RouterGroup) {
	commonapi.RegisterCommonAPIRoutes(r.Group("/commonapi"))
	clusterapi.RegisterUIRoutes(r.Group("/uiapi"))

	// dataapi.RegisterRoutes(r.Group("/dataapi"), st)
}

package server

import (
	"NeuroController/sync/agent/server/commonapi"
	"NeuroController/sync/agent/server/uiapi"

	"github.com/gin-gonic/gin"
)

func RegisterAllAgentRoutes(r *gin.RouterGroup) {
	commonapi.RegisterCommonAPIRoutes(r.Group("/commonapi"))
	uiapi.RegisterUIRoutes(r.Group("/uiapi"))
}

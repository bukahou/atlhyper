package aiservice

import (
	"AtlHyper/atlhyper_master/aiservice/handler"

	"github.com/gin-gonic/gin"
)

func RegisterAISRoutes(rg *gin.RouterGroup) {
	rg.POST("/context/fetch", handler.HandleFetchAIContext)
}

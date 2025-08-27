// ingest/register.go
package ingest

import (
	receivers "NeuroController/external/ingest/receivers"

	"github.com/gin-gonic/gin"
)

func RegisterIngestRoutes(rg *gin.RouterGroup) {
    rg.POST("/events/v1/eventlog", receivers.HandleEventLogIngest)
	rg.POST("/metrics/snapshot", receivers.HandleMetricsSnapshotIngest)
	rg.POST("/podlist", receivers.HandlePodListSnapshotIngest)
}


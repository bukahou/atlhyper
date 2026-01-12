// ingest/register.go
package ingest

import (
	receivers "AtlHyper/atlhyper_master/gateway/handler/ingest/receivers"

	"github.com/gin-gonic/gin"
)

func RegisterIngestRoutes(rg *gin.RouterGroup) {
    rg.POST("/events/v1/eventlog", receivers.HandleEventLogIngest)
	rg.POST("/metrics/snapshot", receivers.HandleMetricsSnapshotIngest)
	rg.POST("/podlist", receivers.HandlePodListSnapshotIngest)
	rg.POST("/nodelist", receivers.HandleNodeListSnapshotIngest)
	rg.POST("/servicelist", receivers.HandleServiceListSnapshotIngest)
	rg.POST("/namespacelist", receivers.HandleNamespaceListSnapshotIngest)
	rg.POST("/ingresslist", receivers.HandleIngressListSnapshotIngest)
	rg.POST("/deploymentlist", receivers.HandleDeploymentListSnapshotIngest)
	rg.POST("/configmaplist", receivers.HandleConfigMapListSnapshotIngest)

}


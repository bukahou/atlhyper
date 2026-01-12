// gateway/routes.go
// API 路径常量定义
package gateway

const (
	PathEventsCleaned   = "/ingest/events/v1/eventlog"
	PathMetricsSnapshot = "/ingest/metrics/snapshot"
	PathPodList         = "/ingest/podlist"
	PathNodeList        = "/ingest/nodelist"
	PathServiceList     = "/ingest/servicelist"
	PathNamespaceList   = "/ingest/namespacelist"
	PathIngressList     = "/ingest/ingresslist"
	PathDeploymentList  = "/ingest/deploymentlist"
	PathConfigMapList   = "/ingest/configmaplist"
	PathOps             = "/ingest/ops"
)
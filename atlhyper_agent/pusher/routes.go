// logic/pusher/routes.go
// 推送路径常量
package pusher

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

// NeuroController/model
package model

// Source 表示数据来源


const (
	SourceK8sEvent        = "k8s_event"
	SourceMetricsSnapshot  = "metrics_snapshot"
	SourcePodListSnapshot = "pod_list_snapshot"
	SourceNodeListSnapshot = "node_list_snapshot"
	SourceServiceListSnapshot = "service_list_snapshot"
	SourceNamespaceListSnapshot = "namespace_list_snapshot"
	SourceIngressListSnapshot   = "ingress_list_snapshot"
	SourceDeploymentListSnapshot = "deployment_list_snapshot"
	SourceConfigMapListSnapshot  = "configmap_list_snapshot"
)

// atlhyper_master_v2/notifier/alert/fingerprint.go
// 指纹：同一个问题始终得到同一个值，去重、恢复通知、重复提醒都以它为准
package alert

// IncidentFingerprint 事件告警的指纹：一个事件一条告警
func IncidentFingerprint(incidentID string) string {
	return string(SourceIncident) + ":" + incidentID
}

// HeartbeatFingerprint Agent 心跳告警的指纹：一个集群一条
func HeartbeatFingerprint(clusterID string) string {
	return string(SourceHeartbeat) + ":" + clusterID
}

// K8sEventFingerprint K8s Warning 事件告警的指纹：同一资源的同一 Reason 一条
func K8sEventFingerprint(clusterID, kind, namespace, name, reason string) string {
	return string(SourceK8sEvent) + ":" + clusterID + ":" + kind + "/" + namespace + "/" + name + ":" + reason
}

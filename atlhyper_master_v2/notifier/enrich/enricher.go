// atlhyper_master_v2/notifier/enrich/enricher.go
// 资源丰富器
// 从 Service Query 获取资源详情，丰富告警信息
package enrich

import (
	"context"
	"fmt"
)

// EnrichedData 丰富后的数据
type EnrichedData struct {
	Pod        *PodInfo
	Node       *NodeInfo
	Deployment *DeploymentInfo
}

// PodInfo Pod 状态信息
type PodInfo struct {
	Phase    string
	Restarts int32
	Ready    string // "1/2"
	NodeName string
}

// NodeInfo Node 状态信息
type NodeInfo struct {
	Ready      bool
	Conditions string
}

// DeploymentInfo Deployment 信息
type DeploymentInfo struct {
	Name      string
	Namespace string
	Replicas  string // "2/3"
}

// Enricher 资源丰富器
type Enricher struct {
	query ResourceQuery
}

// NewEnricher 创建资源丰富器
func NewEnricher(query ResourceQuery) *Enricher {
	return &Enricher{query: query}
}

// EnrichByResource 根据资源类型丰富数据
func (e *Enricher) EnrichByResource(ctx context.Context, clusterID, kind, namespace, name string) *EnrichedData {
	data := &EnrichedData{}

	switch kind {
	case "Pod":
		e.enrichPod(ctx, clusterID, namespace, name, data)
	case "Node":
		e.enrichNode(ctx, clusterID, name, data)
	case "Deployment":
		e.enrichDeployment(ctx, clusterID, namespace, name, data)
	case "ReplicaSet":
		e.enrichFromReplicaSet(ctx, clusterID, namespace, name, data)
	}

	return data
}

// enrichPod 丰富 Pod 信息
func (e *Enricher) enrichPod(ctx context.Context, clusterID, ns, name string, data *EnrichedData) {
	pod, err := e.query.GetPod(ctx, clusterID, ns, name)
	if err != nil || pod == nil {
		return
	}

	data.Pod = &PodInfo{
		Phase:    pod.Status.Phase,
		Restarts: pod.Status.Restarts,
		Ready:    pod.Status.Ready,
		NodeName: pod.Summary.NodeName,
	}

	// 尝试通过 OwnerKind/OwnerName 找到关联的 Deployment
	if pod.Summary.OwnerKind == "ReplicaSet" && pod.Summary.OwnerName != "" {
		e.enrichFromReplicaSet(ctx, clusterID, ns, pod.Summary.OwnerName, data)
	}
}

// enrichNode 丰富 Node 信息
func (e *Enricher) enrichNode(ctx context.Context, clusterID, name string, data *EnrichedData) {
	node, err := e.query.GetNode(ctx, clusterID, name)
	if err != nil || node == nil {
		return
	}

	var ready bool
	var conditions []string
	for _, cond := range node.Conditions {
		if cond.Type == "Ready" {
			ready = cond.Status == "True"
		}
		if cond.Type != "Ready" && cond.Status == "True" {
			conditions = append(conditions, cond.Type)
		}
	}

	condStr := "None"
	if len(conditions) > 0 {
		condStr = fmt.Sprintf("%v", conditions)
	}

	data.Node = &NodeInfo{
		Ready:      ready,
		Conditions: condStr,
	}
}

// enrichDeployment 丰富 Deployment 信息
func (e *Enricher) enrichDeployment(ctx context.Context, clusterID, ns, name string, data *EnrichedData) {
	dep, err := e.query.GetDeployment(ctx, clusterID, ns, name)
	if err != nil || dep == nil {
		return
	}

	data.Deployment = &DeploymentInfo{
		Name:      dep.Summary.Name,
		Namespace: dep.Summary.Namespace,
		Replicas:  fmt.Sprintf("%d/%d", dep.Status.ReadyReplicas, dep.Status.Replicas),
	}
}

// enrichFromReplicaSet 通过 ReplicaSet 找到 Deployment
func (e *Enricher) enrichFromReplicaSet(ctx context.Context, clusterID, ns, rsName string, data *EnrichedData) {
	dep, err := e.query.GetDeploymentByReplicaSet(ctx, clusterID, ns, rsName)
	if err != nil || dep == nil {
		return
	}

	data.Deployment = &DeploymentInfo{
		Name:      dep.Summary.Name,
		Namespace: dep.Summary.Namespace,
		Replicas:  fmt.Sprintf("%d/%d", dep.Status.ReadyReplicas, dep.Status.Replicas),
	}
}

// FormatEnrichedData 格式化丰富数据为字符串（供模板使用）
func FormatEnrichedData(data *EnrichedData) string {
	if data == nil {
		return ""
	}

	var info string

	if data.Pod != nil {
		info += fmt.Sprintf("\nPod 状态:\n  Phase: %s\n  Restarts: %d\n  Ready: %s\n  Node: %s",
			data.Pod.Phase, data.Pod.Restarts, data.Pod.Ready, data.Pod.NodeName)
	}

	if data.Node != nil {
		readyStr := "Yes"
		if !data.Node.Ready {
			readyStr = "No"
		}
		info += fmt.Sprintf("\nNode 状态:\n  Ready: %s\n  Conditions: %s",
			readyStr, data.Node.Conditions)
	}

	if data.Deployment != nil {
		info += fmt.Sprintf("\n关联 Deployment:\n  Name: %s\n  Namespace: %s\n  Replicas: %s",
			data.Deployment.Name, data.Deployment.Namespace, data.Deployment.Replicas)
	}

	return info
}

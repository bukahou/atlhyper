package overview

import (
	"AtlHyper/model/node"
	"AtlHyper/model/pod"
	"strconv"
	"strings"
)

func buildClusterHealth(pods []pod.Pod, nodes []node.Node) ClusterHealthCard {
	// Pod Ready 率
	var podReady, podTotal int
	for _, p := range pods {
		parts := strings.Split(p.Summary.Ready, "/")
		if len(parts) == 2 {
			ready, _ := strconv.Atoi(parts[0])
			total, _ := strconv.Atoi(parts[1])
			podReady += ready
			podTotal += total
		}
	}
	var podPct float64
	if podTotal > 0 {
		podPct = float64(podReady) / float64(podTotal) * 100
	}

	// Node Ready 率
	var nodeReady int
	for _, n := range nodes {
		if strings.ToLower(n.Summary.Ready) == "true" {
			nodeReady++
		}
	}
	nodeTotal := len(nodes)
	var nodePct float64
	if nodeTotal > 0 {
		nodePct = float64(nodeReady) / float64(nodeTotal) * 100
	}

	status := "NoData"
	if nodeTotal > 0 {
		if nodeReady < nodeTotal {
			status = "Degraded"
		} else {
			status = "Healthy"
		}
	}

	return ClusterHealthCard{
		PodReadyPercent:  podPct,
		NodeReadyPercent: nodePct,
		Status:           status,
	}
}

func buildNodeReady(nodes []node.Node) NodeReadyCard {
	total := len(nodes)
	ready := 0
	for _, n := range nodes {
		if strings.ToLower(n.Summary.Ready) == "true" {
			ready++
		}
	}
	var pct float64
	if total > 0 {
		pct = float64(ready) / float64(total) * 100
	}
	return NodeReadyCard{Total: total, Ready: ready, Percent: pct}
}

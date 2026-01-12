package overview

import (
	"strconv"
	"strings"

	"AtlHyper/atlhyper_master/model/dto"
	"AtlHyper/model/k8s"
)

func buildClusterHealth(pods []k8s.Pod, nodes []k8s.Node) dto.ClusterHealthCard {
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

	return dto.ClusterHealthCard{
		PodReadyPercent:  podPct,
		NodeReadyPercent: nodePct,
		Status:           status,
	}
}

func buildNodeReady(nodes []k8s.Node) dto.NodeReadyCard {
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
	return dto.NodeReadyCard{Total: total, Ready: ready, Percent: pct}
}

// atlhyper_master_v2/aiops/baseline/extractor.go
// 从 ClusterSnapshot 和 SLO 数据提取指标数据点
package baseline

import (
	"context"
	"time"

	"AtlHyper/atlhyper_master_v2/aiops"
	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/model_v2"
)

// ExtractMetrics 从快照和 SLO 数据中提取所有实体指标
func ExtractMetrics(
	clusterID string,
	snap *model_v2.ClusterSnapshot,
	sloServiceRepo database.SLOServiceRepository,
	sloRepo database.SLORepository,
) []aiops.MetricDataPoint {
	var points []aiops.MetricDataPoint

	// 1. Node 指标（从 ClusterSnapshot.NodeMetrics）
	points = append(points, extractNodeMetrics(snap)...)

	// 2. Pod 指标（从 K8s 快照）
	points = append(points, extractPodMetrics(snap)...)

	// 3. Service 指标（从 SLO Service Raw — 最近 5 分钟的聚合）
	if sloServiceRepo != nil {
		points = append(points, extractServiceMetrics(clusterID, snap, sloServiceRepo)...)
	}

	// 4. Ingress 指标（从 SLO Metrics Raw）
	if sloRepo != nil {
		points = append(points, extractIngressMetrics(clusterID, snap, sloRepo)...)
	}

	return points
}

func extractNodeMetrics(snap *model_v2.ClusterSnapshot) []aiops.MetricDataPoint {
	var points []aiops.MetricDataPoint
	for nodeName, metrics := range snap.NodeMetrics {
		if metrics == nil {
			continue
		}
		key := aiops.EntityKey("_cluster", "node", nodeName)
		points = append(points,
			aiops.MetricDataPoint{EntityKey: key, MetricName: "cpu_usage", Value: metrics.CPU.UsagePercent},
			aiops.MetricDataPoint{EntityKey: key, MetricName: "memory_usage", Value: metrics.Memory.UsagePercent},
		)
		if disk := metrics.GetPrimaryDisk(); disk != nil {
			points = append(points,
				aiops.MetricDataPoint{EntityKey: key, MetricName: "disk_usage", Value: disk.UsagePercent},
			)
		}
		points = append(points,
			aiops.MetricDataPoint{EntityKey: key, MetricName: "psi_cpu", Value: metrics.PSI.CPUSomePercent},
			aiops.MetricDataPoint{EntityKey: key, MetricName: "psi_memory", Value: metrics.PSI.MemorySomePercent},
			aiops.MetricDataPoint{EntityKey: key, MetricName: "psi_io", Value: metrics.PSI.IOSomePercent},
		)
	}
	return points
}

func extractPodMetrics(snap *model_v2.ClusterSnapshot) []aiops.MetricDataPoint {
	var points []aiops.MetricDataPoint
	for i := range snap.Pods {
		pod := &snap.Pods[i]
		key := aiops.EntityKey(pod.Summary.Namespace, "pod", pod.Summary.Name)
		restarts := float64(pod.Status.Restarts)
		isRunning := 0.0
		if pod.Status.Phase == "Running" {
			isRunning = 1.0
		}
		points = append(points,
			aiops.MetricDataPoint{EntityKey: key, MetricName: "restart_count", Value: restarts},
			aiops.MetricDataPoint{EntityKey: key, MetricName: "is_running", Value: isRunning},
		)
	}
	return points
}

func extractServiceMetrics(clusterID string, snap *model_v2.ClusterSnapshot, sloServiceRepo database.SLOServiceRepository) []aiops.MetricDataPoint {
	var points []aiops.MetricDataPoint
	ctx := context.Background()
	now := time.Now()
	lookback := now.Add(-5 * time.Minute)

	for i := range snap.Services {
		svc := &snap.Services[i]
		key := aiops.EntityKey(svc.Summary.Namespace, "service", svc.Summary.Name)

		raws, err := sloServiceRepo.GetServiceRaw(ctx, clusterID, svc.Summary.Namespace, svc.Summary.Name, lookback, now)
		if err != nil || len(raws) == 0 {
			continue
		}

		var totalReqs, errorReqs int64
		var latencySum float64
		var latencyCount int64
		for _, r := range raws {
			totalReqs += r.TotalRequests
			errorReqs += r.ErrorRequests
			latencySum += r.LatencySum
			latencyCount += r.LatencyCount
		}

		if totalReqs > 0 {
			errorRate := float64(errorReqs) / float64(totalReqs) * 100
			points = append(points,
				aiops.MetricDataPoint{EntityKey: key, MetricName: "error_rate", Value: errorRate},
			)
		}
		if latencyCount > 0 {
			avgLatency := latencySum / float64(latencyCount)
			points = append(points,
				aiops.MetricDataPoint{EntityKey: key, MetricName: "avg_latency", Value: avgLatency},
			)
		}
		if totalReqs > 0 {
			rps := float64(totalReqs) / 300.0 // 5 分钟平均 RPS
			points = append(points,
				aiops.MetricDataPoint{EntityKey: key, MetricName: "request_rate", Value: rps},
			)
		}
	}
	return points
}

func extractIngressMetrics(clusterID string, snap *model_v2.ClusterSnapshot, sloRepo database.SLORepository) []aiops.MetricDataPoint {
	var points []aiops.MetricDataPoint
	ctx := context.Background()
	now := time.Now()
	lookback := now.Add(-5 * time.Minute)

	// 获取所有 hosts
	hosts, err := sloRepo.GetAllHosts(ctx, clusterID)
	if err != nil || len(hosts) == 0 {
		return points
	}

	for _, host := range hosts {
		key := aiops.EntityKey("_cluster", "ingress", host)

		raws, err := sloRepo.GetRawMetrics(ctx, clusterID, host, lookback, now)
		if err != nil || len(raws) == 0 {
			continue
		}

		var totalReqs, errorReqs int64
		var latencySum float64
		var latencyCount int64
		for _, r := range raws {
			totalReqs += r.TotalRequests
			errorReqs += r.ErrorRequests
			latencySum += r.LatencySum
			latencyCount += r.LatencyCount
		}

		if totalReqs > 0 {
			errorRate := float64(errorReqs) / float64(totalReqs) * 100
			points = append(points,
				aiops.MetricDataPoint{EntityKey: key, MetricName: "error_rate", Value: errorRate},
			)
		}
		if latencyCount > 0 {
			avgLatency := latencySum / float64(latencyCount)
			points = append(points,
				aiops.MetricDataPoint{EntityKey: key, MetricName: "avg_latency", Value: avgLatency},
			)
		}
	}
	return points
}

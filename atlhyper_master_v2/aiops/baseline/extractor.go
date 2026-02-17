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

		// 容器级指标
		var notReady float64
		var maxRestarts int32
		for j := range pod.Containers {
			c := &pod.Containers[j]
			if !c.Ready {
				notReady++
			}
			if c.RestartCount > maxRestarts {
				maxRestarts = c.RestartCount
			}
		}

		points = append(points,
			aiops.MetricDataPoint{EntityKey: key, MetricName: "restart_count", Value: restarts},
			aiops.MetricDataPoint{EntityKey: key, MetricName: "is_running", Value: isRunning},
			aiops.MetricDataPoint{EntityKey: key, MetricName: "not_ready_containers", Value: notReady},
			aiops.MetricDataPoint{EntityKey: key, MetricName: "max_container_restarts", Value: float64(maxRestarts)},
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

// ExtractDeterministicAnomalies 从快照中提取确定性异常（绕过 EMA 冷启动）
// 扫描容器状态和关联 Event，对 CrashLoopBackOff/OOMKilled 等确定性异常直接生成 AnomalyResult
func ExtractDeterministicAnomalies(snap *model_v2.ClusterSnapshot) []*aiops.AnomalyResult {
	now := time.Now().Unix()
	var results []*aiops.AnomalyResult

	// 路径 B1: 容器状态异常
	results = append(results, extractContainerAnomalies(snap, now)...)

	// 路径 B2: Event 关联异常
	results = append(results, extractEventAnomalies(snap, now)...)

	return results
}

// extractContainerAnomalies 从容器状态提取确定性异常
// 每个 Pod 只报告最严重的一个容器异常
func extractContainerAnomalies(snap *model_v2.ClusterSnapshot, now int64) []*aiops.AnomalyResult {
	var results []*aiops.AnomalyResult
	for i := range snap.Pods {
		pod := &snap.Pods[i]
		key := aiops.EntityKey(pod.Summary.Namespace, "pod", pod.Summary.Name)

		var worstReason string
		var worstScore float64
		for j := range pod.Containers {
			reason := classifyContainerAnomaly(&pod.Containers[j])
			if reason == "" {
				continue
			}
			score := deterministicScore(reason)
			if score > worstScore {
				worstScore = score
				worstReason = reason
			}
		}

		if worstReason != "" {
			results = append(results, &aiops.AnomalyResult{
				EntityKey:    key,
				MetricName:   "container_anomaly",
				CurrentValue: worstScore,
				Baseline:     0,
				Deviation:    worstScore * 10, // 高偏离度确保触发
				Score:        worstScore,
				IsAnomaly:    true,
				DetectedAt:   now,
			})
		}
	}
	return results
}

// classifyContainerAnomaly 判断容器异常原因
// 返回空字符串表示无异常
func classifyContainerAnomaly(c *model_v2.PodContainerDetail) string {
	// waiting 状态异常（最明确的信号）
	if c.State == "waiting" {
		switch c.StateReason {
		case "CrashLoopBackOff", "OOMKilled",
			"ImagePullBackOff", "ErrImagePull",
			"CreateContainerConfigError":
			return c.StateReason
		}
	}

	// terminated 且因 OOMKilled 终止
	if c.State == "terminated" && c.LastTerminationReason == "OOMKilled" {
		return "OOMKilled"
	}

	// running + 近期崩溃：容器刚重启回来，快照恰好抓到 running 瞬间
	// 检查 LastTerminationTime 在 10 分钟内，避免对历史重启持续告警
	if c.State == "running" && c.RestartCount > 0 && c.LastTerminationReason != "" {
		if isRecentTermination(c.LastTerminationTime) {
			if c.LastTerminationReason == "OOMKilled" {
				return "OOMKilled"
			}
			return "RecentCrash"
		}
	}

	// 就绪探针失败：容器 running 但 Ready=false
	if c.State == "running" && !c.Ready {
		return "NotReady"
	}

	return ""
}

// isRecentTermination 判断上次终止时间是否在 10 分钟内
func isRecentTermination(lastTermTime string) bool {
	if lastTermTime == "" {
		return false
	}
	t, err := time.Parse(time.RFC3339, lastTermTime)
	if err != nil {
		return false
	}
	return time.Since(t) < 10*time.Minute
}

// deterministicScore 异常原因 → 固定分数
func deterministicScore(reason string) float64 {
	switch reason {
	case "OOMKilled":
		return 0.95
	case "CrashLoopBackOff":
		return 0.90
	case "CreateContainerConfigError":
		return 0.80
	case "RecentCrash":
		return 0.75
	case "ImagePullBackOff", "ErrImagePull":
		return 0.70
	case "NotReady":
		return 0.60
	default:
		return 0.50
	}
}

// extractEventAnomalies 从 K8s Event 提取关联异常信号
// 筛选 5 分钟内的 Critical Event，关联到已有 Pod 实体
func extractEventAnomalies(snap *model_v2.ClusterSnapshot, now int64) []*aiops.AnomalyResult {
	cutoff := time.Unix(now, 0).Add(-5 * time.Minute)

	// 构建 Pod 存在性索引
	podExists := make(map[string]bool, len(snap.Pods))
	for i := range snap.Pods {
		pod := &snap.Pods[i]
		key := aiops.EntityKey(pod.Summary.Namespace, "pod", pod.Summary.Name)
		podExists[key] = true
	}

	// 每个 Pod 只报一次
	reported := make(map[string]bool)
	var results []*aiops.AnomalyResult

	for i := range snap.Events {
		ev := &snap.Events[i]
		if !ev.IsCritical() {
			continue
		}
		if ev.InvolvedObject.Kind != "Pod" {
			continue
		}
		if ev.LastTimestamp.Before(cutoff) {
			continue
		}

		key := aiops.EntityKey(ev.InvolvedObject.Namespace, "pod", ev.InvolvedObject.Name)
		if !podExists[key] || reported[key] {
			continue
		}
		reported[key] = true

		results = append(results, &aiops.AnomalyResult{
			EntityKey:    key,
			MetricName:   "critical_event",
			CurrentValue: 0.85,
			Baseline:     0,
			Deviation:    8.5,
			Score:        0.85,
			IsAnomaly:    true,
			DetectedAt:   now,
		})
	}

	return results
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

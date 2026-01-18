// atlhyper_master_v2/query/impl.go
// Query 层实现
package query

import (
	"context"
	"time"

	"AtlHyper/atlhyper_master_v2/datahub"
	"AtlHyper/atlhyper_master_v2/database/repository"
	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/model_v2"
)

// queryImpl Query 接口实现
type queryImpl struct {
	datahub       datahub.DataHub
	eventRepo     repository.ClusterEventRepository
}

// New 创建 Query 实例
func New(dh datahub.DataHub) Query {
	return &queryImpl{
		datahub: dh,
	}
}

// NewWithEventRepo 创建带事件仓库的 Query 实例（用于 Alert Trends）
func NewWithEventRepo(dh datahub.DataHub, eventRepo repository.ClusterEventRepository) Query {
	return &queryImpl{
		datahub:   dh,
		eventRepo: eventRepo,
	}
}

// ==================== 集群查询 ====================

// ListClusters 列出所有集群
func (q *queryImpl) ListClusters(ctx context.Context) ([]model_v2.ClusterInfo, error) {
	agents, err := q.datahub.ListAgents()
	if err != nil {
		return nil, err
	}

	result := make([]model_v2.ClusterInfo, 0, len(agents))
	for _, agent := range agents {
		info := model_v2.ClusterInfo{
			ClusterID: agent.ClusterID,
			Status:    agent.Status,
			LastSeen:  agent.LastHeartbeat,
		}

		// 获取快照统计
		if snapshot, err := q.datahub.GetSnapshot(agent.ClusterID); err == nil && snapshot != nil {
			info.NodeCount = len(snapshot.Nodes)
			info.PodCount = len(snapshot.Pods)
		}

		result = append(result, info)
	}

	return result, nil
}

// GetCluster 获取集群详情
func (q *queryImpl) GetCluster(ctx context.Context, clusterID string) (*model_v2.ClusterDetail, error) {
	snapshot, err := q.GetSnapshot(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	status, _ := q.GetAgentStatus(ctx, clusterID)

	return &model_v2.ClusterDetail{
		ClusterID: clusterID,
		Status:    status,
		Snapshot:  snapshot,
	}, nil
}

// ==================== 快照查询 ====================

// GetSnapshot 获取集群快照
func (q *queryImpl) GetSnapshot(ctx context.Context, clusterID string) (*model_v2.ClusterSnapshot, error) {
	return q.datahub.GetSnapshot(clusterID)
}

// GetPods 获取 Pod 列表
func (q *queryImpl) GetPods(ctx context.Context, clusterID string, opts model.PodQueryOpts) ([]model_v2.Pod, error) {
	snapshot, err := q.datahub.GetSnapshot(clusterID)
	if err != nil || snapshot == nil {
		return nil, err
	}

	// 过滤
	result := make([]model_v2.Pod, 0)
	for _, pod := range snapshot.Pods {
		if opts.Namespace != "" && pod.GetNamespace() != opts.Namespace {
			continue
		}
		if opts.NodeName != "" && pod.GetNodeName() != opts.NodeName {
			continue
		}
		if opts.Phase != "" && pod.Status.Phase != opts.Phase {
			continue
		}

		// 格式化 metrics 单位
		pod.Status.CPUUsage = FormatCPU(pod.Status.CPUUsage)
		pod.Status.MemoryUsage = FormatMemory(pod.Status.MemoryUsage)

		result = append(result, pod)
	}

	// 分页
	if opts.Offset > 0 && opts.Offset < len(result) {
		result = result[opts.Offset:]
	}
	if opts.Limit > 0 && opts.Limit < len(result) {
		result = result[:opts.Limit]
	}

	return result, nil
}

// GetNodes 获取 Node 列表
func (q *queryImpl) GetNodes(ctx context.Context, clusterID string) ([]model_v2.Node, error) {
	snapshot, err := q.datahub.GetSnapshot(clusterID)
	if err != nil || snapshot == nil {
		return nil, err
	}
	return snapshot.Nodes, nil
}

// GetDeployments 获取 Deployment 列表
func (q *queryImpl) GetDeployments(ctx context.Context, clusterID string, namespace string) ([]model_v2.Deployment, error) {
	snapshot, err := q.datahub.GetSnapshot(clusterID)
	if err != nil || snapshot == nil {
		return nil, err
	}

	if namespace == "" {
		return snapshot.Deployments, nil
	}

	result := make([]model_v2.Deployment, 0)
	for _, d := range snapshot.Deployments {
		if d.GetNamespace() == namespace {
			result = append(result, d)
		}
	}
	return result, nil
}

// GetServices 获取 Service 列表
func (q *queryImpl) GetServices(ctx context.Context, clusterID string, namespace string) ([]model_v2.Service, error) {
	snapshot, err := q.datahub.GetSnapshot(clusterID)
	if err != nil || snapshot == nil {
		return nil, err
	}

	if namespace == "" {
		return snapshot.Services, nil
	}

	result := make([]model_v2.Service, 0)
	for _, s := range snapshot.Services {
		if s.GetNamespace() == namespace {
			result = append(result, s)
		}
	}
	return result, nil
}

// GetIngresses 获取 Ingress 列表
func (q *queryImpl) GetIngresses(ctx context.Context, clusterID string, namespace string) ([]model_v2.Ingress, error) {
	snapshot, err := q.datahub.GetSnapshot(clusterID)
	if err != nil || snapshot == nil {
		return nil, err
	}

	if namespace == "" {
		return snapshot.Ingresses, nil
	}

	result := make([]model_v2.Ingress, 0)
	for _, i := range snapshot.Ingresses {
		if i.GetNamespace() == namespace {
			result = append(result, i)
		}
	}
	return result, nil
}

// GetConfigMaps 获取 ConfigMap 列表
func (q *queryImpl) GetConfigMaps(ctx context.Context, clusterID string, namespace string) ([]model_v2.ConfigMap, error) {
	snapshot, err := q.datahub.GetSnapshot(clusterID)
	if err != nil || snapshot == nil {
		return nil, err
	}

	if namespace == "" {
		return snapshot.ConfigMaps, nil
	}

	result := make([]model_v2.ConfigMap, 0)
	for _, c := range snapshot.ConfigMaps {
		if c.Namespace == namespace {
			result = append(result, c)
		}
	}
	return result, nil
}

// GetSecrets 获取 Secret 列表
func (q *queryImpl) GetSecrets(ctx context.Context, clusterID string, namespace string) ([]model_v2.Secret, error) {
	snapshot, err := q.datahub.GetSnapshot(clusterID)
	if err != nil || snapshot == nil {
		return nil, err
	}

	if namespace == "" {
		return snapshot.Secrets, nil
	}

	result := make([]model_v2.Secret, 0)
	for _, s := range snapshot.Secrets {
		if s.Namespace == namespace {
			result = append(result, s)
		}
	}
	return result, nil
}

// GetNamespaces 获取 Namespace 列表
func (q *queryImpl) GetNamespaces(ctx context.Context, clusterID string) ([]model_v2.Namespace, error) {
	snapshot, err := q.datahub.GetSnapshot(clusterID)
	if err != nil || snapshot == nil {
		return nil, err
	}
	return snapshot.Namespaces, nil
}

// GetDaemonSets 获取 DaemonSet 列表
func (q *queryImpl) GetDaemonSets(ctx context.Context, clusterID string, namespace string) ([]model_v2.DaemonSet, error) {
	snapshot, err := q.datahub.GetSnapshot(clusterID)
	if err != nil || snapshot == nil {
		return nil, err
	}

	if namespace == "" {
		return snapshot.DaemonSets, nil
	}

	result := make([]model_v2.DaemonSet, 0)
	for _, d := range snapshot.DaemonSets {
		if d.GetNamespace() == namespace {
			result = append(result, d)
		}
	}
	return result, nil
}

// GetStatefulSets 获取 StatefulSet 列表
func (q *queryImpl) GetStatefulSets(ctx context.Context, clusterID string, namespace string) ([]model_v2.StatefulSet, error) {
	snapshot, err := q.datahub.GetSnapshot(clusterID)
	if err != nil || snapshot == nil {
		return nil, err
	}

	if namespace == "" {
		return snapshot.StatefulSets, nil
	}

	result := make([]model_v2.StatefulSet, 0)
	for _, s := range snapshot.StatefulSets {
		if s.GetNamespace() == namespace {
			result = append(result, s)
		}
	}
	return result, nil
}

// ==================== Event 查询 ====================

// GetEvents 获取实时 Events
func (q *queryImpl) GetEvents(ctx context.Context, clusterID string, opts model.EventQueryOpts) ([]model_v2.Event, error) {
	events, err := q.datahub.GetEvents(clusterID)
	if err != nil {
		return nil, err
	}

	result := make([]model_v2.Event, 0, len(events))
	for _, e := range events {
		// 过滤
		if opts.Type != "" && e.Type != opts.Type {
			continue
		}
		if opts.Reason != "" && e.Reason != opts.Reason {
			continue
		}
		if !opts.Since.IsZero() && e.LastTimestamp.Before(opts.Since) {
			continue
		}
		result = append(result, e)
	}

	// 分页
	if opts.Offset > 0 && opts.Offset < len(result) {
		result = result[opts.Offset:]
	}
	if opts.Limit > 0 && opts.Limit < len(result) {
		result = result[:opts.Limit]
	}

	return result, nil
}

// GetEventsByResource 按资源查询 Events
func (q *queryImpl) GetEventsByResource(ctx context.Context, clusterID, kind, namespace, name string) ([]model_v2.Event, error) {
	events, err := q.datahub.GetEvents(clusterID)
	if err != nil {
		return nil, err
	}

	result := make([]model_v2.Event, 0)
	for _, e := range events {
		if e.InvolvedObject.Kind == kind && e.InvolvedObject.Namespace == namespace && e.InvolvedObject.Name == name {
			result = append(result, e)
		}
	}

	return result, nil
}

// ==================== Agent 状态查询 ====================

// GetAgentStatus 获取 Agent 状态
func (q *queryImpl) GetAgentStatus(ctx context.Context, clusterID string) (*model_v2.AgentStatus, error) {
	return q.datahub.GetAgentStatus(clusterID)
}

// ==================== 指令状态查询 ====================

// GetCommandStatus 获取指令状态
func (q *queryImpl) GetCommandStatus(ctx context.Context, commandID string) (*model.CommandStatus, error) {
	return q.datahub.GetCommandStatus(commandID)
}

// ==================== 概览查询 ====================

// GetOverview 获取集群概览
func (q *queryImpl) GetOverview(ctx context.Context, clusterID string) (*model_v2.ClusterOverview, error) {
	// 获取快照
	snapshot, err := q.datahub.GetSnapshot(clusterID)
	if err != nil {
		return nil, err
	}
	if snapshot == nil {
		return nil, nil
	}

	// 使用 model_v2 的 Summary
	summary := snapshot.Summary

	// 计算百分比
	nodeReadyPct := 0.0
	if summary.TotalNodes > 0 {
		nodeReadyPct = float64(summary.ReadyNodes) / float64(summary.TotalNodes) * 100
	}

	podReadyPct := 0.0
	if summary.TotalPods > 0 {
		podReadyPct = float64(summary.RunningPods) / float64(summary.TotalPods) * 100
	}

	// 构建健康状态
	healthStatus := model_v2.CalculateHealthStatus(nodeReadyPct, podReadyPct)
	healthReason := model_v2.CalculateHealthReason(nodeReadyPct, podReadyPct)

	// 计算集群 CPU/Memory 使用率
	var totalCPUAllocatable, totalCPUUsage int64
	var totalMemAllocatable, totalMemUsage int64
	nodeUsages := make([]model_v2.NodeUsage, 0, len(snapshot.Nodes))
	var peakCPU, peakMem float64
	var peakCPUNode, peakMemNode string

	for _, node := range snapshot.Nodes {
		cpuAllocatable := model_v2.ParseCPU(node.Allocatable.CPU)
		memAllocatable := model_v2.ParseMemory(node.Allocatable.Memory)

		// 从 Metrics 获取使用量
		var cpuUsage, memUsage int64
		if node.Metrics != nil {
			cpuUsage = model_v2.ParseCPU(node.Metrics.CPU.Usage)
			memUsage = model_v2.ParseMemory(node.Metrics.Memory.Usage)
		}

		totalCPUAllocatable += cpuAllocatable
		totalCPUUsage += cpuUsage
		totalMemAllocatable += memAllocatable
		totalMemUsage += memUsage

		// 计算单节点使用率
		nodeCPUPct := 0.0
		nodeMemPct := 0.0
		if cpuAllocatable > 0 {
			nodeCPUPct = float64(cpuUsage) / float64(cpuAllocatable) * 100
		}
		if memAllocatable > 0 {
			nodeMemPct = float64(memUsage) / float64(memAllocatable) * 100
		}

		// 只添加有 metrics 数据的节点
		if cpuUsage > 0 || memUsage > 0 {
			nodeUsages = append(nodeUsages, model_v2.NodeUsage{
				Node:     node.GetName(),
				CPUUsage: nodeCPUPct,
				MemUsage: nodeMemPct,
			})

			// 记录峰值
			if nodeCPUPct > peakCPU {
				peakCPU = nodeCPUPct
				peakCPUNode = node.GetName()
			}
			if nodeMemPct > peakMem {
				peakMem = nodeMemPct
				peakMemNode = node.GetName()
			}
		}
	}

	// 计算集群总使用率
	clusterCPUPct := 0.0
	clusterMemPct := 0.0
	if totalCPUAllocatable > 0 {
		clusterCPUPct = float64(totalCPUUsage) / float64(totalCPUAllocatable) * 100
	}
	if totalMemAllocatable > 0 {
		clusterMemPct = float64(totalMemUsage) / float64(totalMemAllocatable) * 100
	}

	hasMetrics := len(nodeUsages) > 0

	// ==================== 告警数据（全部从数据库获取）====================
	recentAlerts := make([]model_v2.RecentAlert, 0)
	alertTotal := 0

	// 1. 告警趋势（完整的 24 小时时间线，按资源类型统计）
	// 时间范围：从当前小时向前推 23 小时，共 24 个点
	now := time.Now()
	currentHour := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, now.Location())

	// 初始化 24 个时间点，每个时间点的 Kinds 为空 map
	alertTrend := make([]model_v2.AlertTrendPoint, 24)
	hourToIndex := make(map[string]int)
	for i := 0; i < 24; i++ {
		t := currentHour.Add(time.Duration(i-23) * time.Hour)
		alertTrend[i] = model_v2.AlertTrendPoint{
			At:    t,
			Kinds: make(map[string]int),
		}
		hourToIndex[t.Format("2006-01-02T15")] = i
	}

	if q.eventRepo != nil {
		// 从数据库获取按小时和资源类型统计的数据
		hourlyKindStats, _ := q.eventRepo.CountByHourAndKind(ctx, clusterID, 24)
		for _, h := range hourlyKindStats {
			if idx, ok := hourToIndex[h.Hour]; ok {
				alertTrend[idx].Kinds[h.Kind] = h.Count
				alertTotal += h.Count
			}
		}

		// 2. 最近告警（最近 10 条 Warning 事件）
		since24h := now.Add(-24 * time.Hour)
		dbEvents, _ := q.eventRepo.ListByCluster(ctx, clusterID, repository.EventQueryOpts{
			Since: since24h,
			Limit: 10,
		})
		for _, e := range dbEvents {
			recentAlerts = append(recentAlerts, model_v2.RecentAlert{
				Timestamp: e.LastTimestamp.Format("2006-01-02T15:04:05Z"),
				Severity:  "warning", // 数据库只存 Warning
				Kind:      e.InvolvedKind,
				Namespace: e.InvolvedNamespace,
				Name:      e.InvolvedName,
				Message:   e.Message,
				Reason:    e.Reason,
			})
		}
	}

	// ==================== 计算工作负载统计 ====================

	// Deployments 统计
	deploymentTotal := len(snapshot.Deployments)
	deploymentReady := 0
	for _, d := range snapshot.Deployments {
		if d.IsHealthy() {
			deploymentReady++
		}
	}

	// StatefulSets 统计
	statefulSetTotal := len(snapshot.StatefulSets)
	statefulSetReady := 0
	for _, s := range snapshot.StatefulSets {
		if s.IsHealthy() {
			statefulSetReady++
		}
	}

	// DaemonSets 统计
	daemonSetTotal := len(snapshot.DaemonSets)
	daemonSetReady := 0
	for _, d := range snapshot.DaemonSets {
		if d.IsHealthy() {
			daemonSetReady++
		}
	}

	// Jobs 统计
	jobTotal := len(snapshot.Jobs)
	jobRunning := 0
	jobSucceeded := 0
	jobFailed := 0
	for _, j := range snapshot.Jobs {
		if j.Active > 0 {
			jobRunning++
		}
		if j.Complete && j.Succeeded > 0 {
			jobSucceeded++
		}
		if j.Failed > 0 {
			jobFailed++
		}
	}

	// Pod 状态分布
	podTotal := len(snapshot.Pods)
	podRunning := 0
	podPending := 0
	podFailed := 0
	podSucceeded := 0
	podUnknown := 0
	for _, p := range snapshot.Pods {
		switch p.Status.Phase {
		case "Running":
			podRunning++
		case "Pending":
			podPending++
		case "Failed":
			podFailed++
		case "Succeeded":
			podSucceeded++
		default:
			podUnknown++
		}
	}

	// 计算百分比
	runningPct := 0.0
	pendingPct := 0.0
	failedPct := 0.0
	succeededPct := 0.0
	if podTotal > 0 {
		runningPct = float64(podRunning) / float64(podTotal) * 100
		pendingPct = float64(podPending) / float64(podTotal) * 100
		failedPct = float64(podFailed) / float64(podTotal) * 100
		succeededPct = float64(podSucceeded) / float64(podTotal) * 100
	}

	// 构建响应
	overview := &model_v2.ClusterOverview{
		ClusterID: clusterID,
		Cards: model_v2.OverviewCards{
			ClusterHealth: model_v2.ClusterHealth{
				Status:           healthStatus,
				Reason:           healthReason,
				NodeReadyPercent: nodeReadyPct,
				PodReadyPercent:  podReadyPct,
			},
			NodeReady: model_v2.ResourceReady{
				Total:   summary.TotalNodes,
				Ready:   summary.ReadyNodes,
				Percent: nodeReadyPct,
			},
			CPUUsage:  model_v2.ResourcePercent{Percent: clusterCPUPct},
			MemUsage:  model_v2.ResourcePercent{Percent: clusterMemPct},
			Events24h: alertTotal,
		},
		Workloads: model_v2.OverviewWorkloads{
			Summary: model_v2.WorkloadSummary{
				Deployments:  model_v2.WorkloadStatus{Total: deploymentTotal, Ready: deploymentReady},
				DaemonSets:   model_v2.WorkloadStatus{Total: daemonSetTotal, Ready: daemonSetReady},
				StatefulSets: model_v2.WorkloadStatus{Total: statefulSetTotal, Ready: statefulSetReady},
				Jobs:         model_v2.JobStatus{Total: jobTotal, Running: jobRunning, Succeeded: jobSucceeded, Failed: jobFailed},
			},
			PodStatus: model_v2.PodStatusDistribution{
				Total:            podTotal,
				Running:          podRunning,
				Pending:          podPending,
				Failed:           podFailed,
				Succeeded:        podSucceeded,
				Unknown:          podUnknown,
				RunningPercent:   runningPct,
				PendingPercent:   pendingPct,
				FailedPercent:    failedPct,
				SucceededPercent: succeededPct,
			},
			PeakStats: &model_v2.PeakStats{
				PeakCPU:     peakCPU,
				PeakCPUNode: peakCPUNode,
				PeakMem:     peakMem,
				PeakMemNode: peakMemNode,
				HasData:     hasMetrics,
			},
		},
		Alerts: model_v2.OverviewAlerts{
			Trend: alertTrend,
			Totals: model_v2.AlertTotals{
				Critical: 0,           // 不再区分严重程度
				Warning:  alertTotal,  // 全部告警数
				Info:     0,
			},
			Recent: recentAlerts,
		},
		Nodes: model_v2.OverviewNodes{
			Usage: nodeUsages,
		},
	}

	return overview, nil
}

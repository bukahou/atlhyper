// atlhyper_master_v2/query/impl.go
// Query 层实现
package query

import (
	"context"
	"time"

	"AtlHyper/atlhyper_master_v2/aiops"
	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/datahub"
	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/atlhyper_master_v2/mq"
	"AtlHyper/model_v2"
)

// QueryService Query 层实现
type QueryService struct {
	store       datahub.Store
	bus         mq.Producer
	eventRepo   database.ClusterEventRepository
	sloRepo     database.SLORepository
	serviceRepo database.SLOServiceRepository
	edgeRepo    database.SLOEdgeRepository
	aiopsEngine aiops.Engine
}

// New 创建 QueryService 实例
func New(store datahub.Store, bus mq.Producer) *QueryService {
	return &QueryService{
		store: store,
		bus:   bus,
	}
}

// NewWithEventRepo 创建带事件仓库的 QueryService 实例（用于 Alert Trends）
func NewWithEventRepo(store datahub.Store, bus mq.Producer, eventRepo database.ClusterEventRepository) *QueryService {
	return &QueryService{
		store:     store,
		bus:       bus,
		eventRepo: eventRepo,
	}
}

// NewWithSLORepos 创建带 SLO 仓库的 QueryService 实例
func NewWithSLORepos(store datahub.Store, bus mq.Producer, eventRepo database.ClusterEventRepository, sloRepo database.SLORepository, serviceRepo database.SLOServiceRepository, edgeRepo database.SLOEdgeRepository) *QueryService {
	return &QueryService{
		store:       store,
		bus:         bus,
		eventRepo:   eventRepo,
		sloRepo:     sloRepo,
		serviceRepo: serviceRepo,
		edgeRepo:    edgeRepo,
	}
}

// SetAIOpsEngine 注入 AIOps 引擎（可选）
func (q *QueryService) SetAIOpsEngine(engine aiops.Engine) {
	q.aiopsEngine = engine
}

// ==================== 集群查询 ====================

// ListClusters 列出所有集群
func (q *QueryService) ListClusters(ctx context.Context) ([]model_v2.ClusterInfo, error) {
	agents, err := q.store.ListAgents()
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
		if snapshot, err := q.store.GetSnapshot(agent.ClusterID); err == nil && snapshot != nil {
			info.NodeCount = len(snapshot.Nodes)
			info.PodCount = len(snapshot.Pods)
		}

		result = append(result, info)
	}

	return result, nil
}

// GetCluster 获取集群详情
func (q *QueryService) GetCluster(ctx context.Context, clusterID string) (*model_v2.ClusterDetail, error) {
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
func (q *QueryService) GetSnapshot(ctx context.Context, clusterID string) (*model_v2.ClusterSnapshot, error) {
	return q.store.GetSnapshot(clusterID)
}

// GetPods 获取 Pod 列表
func (q *QueryService) GetPods(ctx context.Context, clusterID string, opts model.PodQueryOpts) ([]model_v2.Pod, error) {
	snapshot, err := q.store.GetSnapshot(clusterID)
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
func (q *QueryService) GetNodes(ctx context.Context, clusterID string) ([]model_v2.Node, error) {
	snapshot, err := q.store.GetSnapshot(clusterID)
	if err != nil || snapshot == nil {
		return nil, err
	}
	return snapshot.Nodes, nil
}

// GetDeployments 获取 Deployment 列表
func (q *QueryService) GetDeployments(ctx context.Context, clusterID string, namespace string) ([]model_v2.Deployment, error) {
	snapshot, err := q.store.GetSnapshot(clusterID)
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
func (q *QueryService) GetServices(ctx context.Context, clusterID string, namespace string) ([]model_v2.Service, error) {
	snapshot, err := q.store.GetSnapshot(clusterID)
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
func (q *QueryService) GetIngresses(ctx context.Context, clusterID string, namespace string) ([]model_v2.Ingress, error) {
	snapshot, err := q.store.GetSnapshot(clusterID)
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
func (q *QueryService) GetConfigMaps(ctx context.Context, clusterID string, namespace string) ([]model_v2.ConfigMap, error) {
	snapshot, err := q.store.GetSnapshot(clusterID)
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
func (q *QueryService) GetSecrets(ctx context.Context, clusterID string, namespace string) ([]model_v2.Secret, error) {
	snapshot, err := q.store.GetSnapshot(clusterID)
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
func (q *QueryService) GetNamespaces(ctx context.Context, clusterID string) ([]model_v2.Namespace, error) {
	snapshot, err := q.store.GetSnapshot(clusterID)
	if err != nil || snapshot == nil {
		return nil, err
	}
	return snapshot.Namespaces, nil
}

// GetDaemonSets 获取 DaemonSet 列表
func (q *QueryService) GetDaemonSets(ctx context.Context, clusterID string, namespace string) ([]model_v2.DaemonSet, error) {
	snapshot, err := q.store.GetSnapshot(clusterID)
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
func (q *QueryService) GetStatefulSets(ctx context.Context, clusterID string, namespace string) ([]model_v2.StatefulSet, error) {
	snapshot, err := q.store.GetSnapshot(clusterID)
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

// ==================== Job / CronJob 查询 ====================

// GetJobs 获取 Job 列表
func (q *QueryService) GetJobs(ctx context.Context, clusterID string, namespace string) ([]model_v2.Job, error) {
	snapshot, err := q.store.GetSnapshot(clusterID)
	if err != nil || snapshot == nil {
		return nil, err
	}

	if namespace == "" {
		return snapshot.Jobs, nil
	}

	result := make([]model_v2.Job, 0)
	for _, j := range snapshot.Jobs {
		if j.Namespace == namespace {
			result = append(result, j)
		}
	}
	return result, nil
}

// GetCronJobs 获取 CronJob 列表
func (q *QueryService) GetCronJobs(ctx context.Context, clusterID string, namespace string) ([]model_v2.CronJob, error) {
	snapshot, err := q.store.GetSnapshot(clusterID)
	if err != nil || snapshot == nil {
		return nil, err
	}

	if namespace == "" {
		return snapshot.CronJobs, nil
	}

	result := make([]model_v2.CronJob, 0)
	for _, c := range snapshot.CronJobs {
		if c.Namespace == namespace {
			result = append(result, c)
		}
	}
	return result, nil
}

// ==================== 存储查询 ====================

// GetPersistentVolumes 获取 PV 列表（集群级，无 namespace）
func (q *QueryService) GetPersistentVolumes(ctx context.Context, clusterID string) ([]model_v2.PersistentVolume, error) {
	snapshot, err := q.store.GetSnapshot(clusterID)
	if err != nil || snapshot == nil {
		return nil, err
	}
	return snapshot.PersistentVolumes, nil
}

// GetPersistentVolumeClaims 获取 PVC 列表
func (q *QueryService) GetPersistentVolumeClaims(ctx context.Context, clusterID string, namespace string) ([]model_v2.PersistentVolumeClaim, error) {
	snapshot, err := q.store.GetSnapshot(clusterID)
	if err != nil || snapshot == nil {
		return nil, err
	}

	if namespace == "" {
		return snapshot.PersistentVolumeClaims, nil
	}

	result := make([]model_v2.PersistentVolumeClaim, 0)
	for _, p := range snapshot.PersistentVolumeClaims {
		if p.Namespace == namespace {
			result = append(result, p)
		}
	}
	return result, nil
}

// ==================== 策略与配额查询 ====================

// GetNetworkPolicies 获取 NetworkPolicy 列表
func (q *QueryService) GetNetworkPolicies(ctx context.Context, clusterID string, namespace string) ([]model_v2.NetworkPolicy, error) {
	snapshot, err := q.store.GetSnapshot(clusterID)
	if err != nil || snapshot == nil {
		return nil, err
	}

	if namespace == "" {
		return snapshot.NetworkPolicies, nil
	}

	result := make([]model_v2.NetworkPolicy, 0)
	for _, np := range snapshot.NetworkPolicies {
		if np.Namespace == namespace {
			result = append(result, np)
		}
	}
	return result, nil
}

// GetResourceQuotas 获取 ResourceQuota 列表
func (q *QueryService) GetResourceQuotas(ctx context.Context, clusterID string, namespace string) ([]model_v2.ResourceQuota, error) {
	snapshot, err := q.store.GetSnapshot(clusterID)
	if err != nil || snapshot == nil {
		return nil, err
	}

	if namespace == "" {
		return snapshot.ResourceQuotas, nil
	}

	result := make([]model_v2.ResourceQuota, 0)
	for _, rq := range snapshot.ResourceQuotas {
		if rq.Namespace == namespace {
			result = append(result, rq)
		}
	}
	return result, nil
}

// GetLimitRanges 获取 LimitRange 列表
func (q *QueryService) GetLimitRanges(ctx context.Context, clusterID string, namespace string) ([]model_v2.LimitRange, error) {
	snapshot, err := q.store.GetSnapshot(clusterID)
	if err != nil || snapshot == nil {
		return nil, err
	}

	if namespace == "" {
		return snapshot.LimitRanges, nil
	}

	result := make([]model_v2.LimitRange, 0)
	for _, lr := range snapshot.LimitRanges {
		if lr.Namespace == namespace {
			result = append(result, lr)
		}
	}
	return result, nil
}

// GetServiceAccounts 获取 ServiceAccount 列表
func (q *QueryService) GetServiceAccounts(ctx context.Context, clusterID string, namespace string) ([]model_v2.ServiceAccount, error) {
	snapshot, err := q.store.GetSnapshot(clusterID)
	if err != nil || snapshot == nil {
		return nil, err
	}

	if namespace == "" {
		return snapshot.ServiceAccounts, nil
	}

	result := make([]model_v2.ServiceAccount, 0)
	for _, sa := range snapshot.ServiceAccounts {
		if sa.Namespace == namespace {
			result = append(result, sa)
		}
	}
	return result, nil
}

// ==================== Event 查询 ====================

// GetEvents 获取实时 Events
func (q *QueryService) GetEvents(ctx context.Context, clusterID string, opts model.EventQueryOpts) ([]model_v2.Event, error) {
	events, err := q.store.GetEvents(clusterID)
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
func (q *QueryService) GetEventsByResource(ctx context.Context, clusterID, kind, namespace, name string) ([]model_v2.Event, error) {
	events, err := q.store.GetEvents(clusterID)
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
func (q *QueryService) GetAgentStatus(ctx context.Context, clusterID string) (*model_v2.AgentStatus, error) {
	return q.store.GetAgentStatus(clusterID)
}

// ==================== 指令状态查询 ====================

// GetCommandStatus 获取指令状态
func (q *QueryService) GetCommandStatus(ctx context.Context, commandID string) (*model.CommandStatus, error) {
	return q.bus.GetCommandStatus(commandID)
}

// ==================== 概览查询 ====================

// GetOverview 获取集群概览
func (q *QueryService) GetOverview(ctx context.Context, clusterID string) (*model_v2.ClusterOverview, error) {
	// 获取快照
	snapshot, err := q.store.GetSnapshot(clusterID)
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
		dbEvents, _ := q.eventRepo.ListByCluster(ctx, clusterID, database.EventQueryOpts{
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

// ==================== 单资源查询 (Event Alert Enrichment) ====================

// GetPod 获取单个 Pod
func (q *QueryService) GetPod(ctx context.Context, clusterID, namespace, name string) (*model_v2.Pod, error) {
	snapshot, err := q.store.GetSnapshot(clusterID)
	if err != nil || snapshot == nil {
		return nil, err
	}

	for i := range snapshot.Pods {
		pod := &snapshot.Pods[i]
		if pod.GetNamespace() == namespace && pod.GetName() == name {
			return pod, nil
		}
	}
	return nil, nil
}

// GetNode 获取单个 Node
func (q *QueryService) GetNode(ctx context.Context, clusterID, name string) (*model_v2.Node, error) {
	snapshot, err := q.store.GetSnapshot(clusterID)
	if err != nil || snapshot == nil {
		return nil, err
	}

	for i := range snapshot.Nodes {
		node := &snapshot.Nodes[i]
		if node.GetName() == name {
			return node, nil
		}
	}
	return nil, nil
}

// GetDeployment 获取单个 Deployment
func (q *QueryService) GetDeployment(ctx context.Context, clusterID, namespace, name string) (*model_v2.Deployment, error) {
	snapshot, err := q.store.GetSnapshot(clusterID)
	if err != nil || snapshot == nil {
		return nil, err
	}

	for i := range snapshot.Deployments {
		dep := &snapshot.Deployments[i]
		if dep.GetNamespace() == namespace && dep.GetName() == name {
			return dep, nil
		}
	}
	return nil, nil
}

// GetDeploymentByReplicaSet 通过 ReplicaSet 名称查找所属 Deployment
// ReplicaSet 名称格式: {deployment-name}-{hash}
func (q *QueryService) GetDeploymentByReplicaSet(ctx context.Context, clusterID, namespace, rsName string) (*model_v2.Deployment, error) {
	snapshot, err := q.store.GetSnapshot(clusterID)
	if err != nil || snapshot == nil {
		return nil, err
	}

	// ReplicaSet 名称格式: {deployment-name}-{hash}
	// 遍历 Deployment 查找匹配的
	for i := range snapshot.Deployments {
		dep := &snapshot.Deployments[i]
		if dep.GetNamespace() != namespace {
			continue
		}

		// 检查 ReplicaSet 名称是否以 Deployment 名称为前缀
		depName := dep.GetName()
		if len(rsName) > len(depName)+1 && rsName[:len(depName)+1] == depName+"-" {
			return dep, nil
		}
	}
	return nil, nil
}

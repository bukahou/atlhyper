// Package snapshot 集群快照采集服务
//
// 本包实现 service.SnapshotService 接口，负责:
//   - 并发采集集群中的所有资源 (20 种类型)
//   - 生成集群统计摘要 (ClusterSummary)
//   - 组装完整的 ClusterSnapshot 对象
//
// 采集策略:
//   - 使用 goroutine 并发采集各类资源，提高效率
//   - 使用 sync.Mutex 保护共享的 snapshot 对象
//   - 只记录第一个错误，即使部分资源采集失败也返回已采集的数据
package snapshot

import (
	"context"
	"sync"
	"time"

	"AtlHyper/atlhyper_agent_v2/model"
	"AtlHyper/atlhyper_agent_v2/repository"
	"AtlHyper/atlhyper_agent_v2/service"
	"AtlHyper/model_v2"
)

// snapshotService 快照采集服务实现
//
// 依赖 20 个 K8s Repository + 可选的 SLO Repository。
// 所有 Repository 在创建时注入，支持测试时 mock。
type snapshotService struct {
	clusterID string

	podRepo            repository.PodRepository
	nodeRepo           repository.NodeRepository
	deploymentRepo     repository.DeploymentRepository
	statefulSetRepo    repository.StatefulSetRepository
	daemonSetRepo      repository.DaemonSetRepository
	replicaSetRepo     repository.ReplicaSetRepository
	serviceRepo        repository.ServiceRepository
	ingressRepo        repository.IngressRepository
	configMapRepo      repository.ConfigMapRepository
	secretRepo         repository.SecretRepository
	namespaceRepo      repository.NamespaceRepository
	eventRepo          repository.EventRepository
	jobRepo            repository.JobRepository
	cronJobRepo        repository.CronJobRepository
	pvRepo             repository.PersistentVolumeRepository
	pvcRepo            repository.PersistentVolumeClaimRepository
	resourceQuotaRepo  repository.ResourceQuotaRepository
	limitRangeRepo     repository.LimitRangeRepository
	networkPolicyRepo  repository.NetworkPolicyRepository
	serviceAccountRepo repository.ServiceAccountRepository

	// 节点指标仓库 (可选)
	metricsRepo repository.MetricsRepository

	// SLO 数据仓库 (可选)
	sloRepo repository.SLORepository
}

// NewSnapshotService 创建快照服务
func NewSnapshotService(
	clusterID string,
	podRepo repository.PodRepository,
	nodeRepo repository.NodeRepository,
	deploymentRepo repository.DeploymentRepository,
	statefulSetRepo repository.StatefulSetRepository,
	daemonSetRepo repository.DaemonSetRepository,
	replicaSetRepo repository.ReplicaSetRepository,
	serviceRepo repository.ServiceRepository,
	ingressRepo repository.IngressRepository,
	configMapRepo repository.ConfigMapRepository,
	secretRepo repository.SecretRepository,
	namespaceRepo repository.NamespaceRepository,
	eventRepo repository.EventRepository,
	jobRepo repository.JobRepository,
	cronJobRepo repository.CronJobRepository,
	pvRepo repository.PersistentVolumeRepository,
	pvcRepo repository.PersistentVolumeClaimRepository,
	resourceQuotaRepo repository.ResourceQuotaRepository,
	limitRangeRepo repository.LimitRangeRepository,
	networkPolicyRepo repository.NetworkPolicyRepository,
	serviceAccountRepo repository.ServiceAccountRepository,
	metricsRepo repository.MetricsRepository,
	sloRepo repository.SLORepository,
) service.SnapshotService {
	return &snapshotService{
		clusterID:          clusterID,
		podRepo:            podRepo,
		nodeRepo:           nodeRepo,
		deploymentRepo:     deploymentRepo,
		statefulSetRepo:    statefulSetRepo,
		daemonSetRepo:      daemonSetRepo,
		replicaSetRepo:     replicaSetRepo,
		serviceRepo:        serviceRepo,
		ingressRepo:        ingressRepo,
		configMapRepo:      configMapRepo,
		secretRepo:         secretRepo,
		namespaceRepo:      namespaceRepo,
		eventRepo:          eventRepo,
		jobRepo:            jobRepo,
		cronJobRepo:        cronJobRepo,
		pvRepo:             pvRepo,
		pvcRepo:            pvcRepo,
		resourceQuotaRepo:  resourceQuotaRepo,
		limitRangeRepo:     limitRangeRepo,
		networkPolicyRepo:  networkPolicyRepo,
		serviceAccountRepo: serviceAccountRepo,
		metricsRepo:        metricsRepo,
		sloRepo:            sloRepo,
	}
}

// Collect 采集集群快照
//
// 并发采集 20 种 K8s 资源，组装为完整的 ClusterSnapshot。
// 采集完成后生成统计摘要，用于仪表盘快速展示。
//
// 并发策略:
//   - 启动 20 个 goroutine 同时采集
//   - 使用 WaitGroup 等待全部完成
//   - 使用 Mutex 保护 snapshot 写入
//
// 错误处理:
//   - 记录第一个错误但不中断其他采集
//   - 返回已成功采集的数据
func (s *snapshotService) Collect(ctx context.Context) (*model_v2.ClusterSnapshot, error) {
	snapshot := &model_v2.ClusterSnapshot{
		ClusterID: s.clusterID,
		FetchedAt: time.Now(),
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error

	// 记录第一个错误
	recordErr := func(err error) {
		if err != nil {
			mu.Lock()
			if firstErr == nil {
				firstErr = err
			}
			mu.Unlock()
		}
	}

	opts := model.ListOptions{}

	// 计算并发任务数: 20 个 K8s 资源 + 可选的 SLO
	taskCount := 20
	if s.sloRepo != nil {
		taskCount++
	}
	wg.Add(taskCount)

	// Pods
	go func() {
		defer wg.Done()
		pods, err := s.podRepo.List(ctx, "", opts)
		recordErr(err)
		if err == nil {
			mu.Lock()
			snapshot.Pods = pods
			mu.Unlock()
		}
	}()

	// Nodes
	go func() {
		defer wg.Done()
		nodes, err := s.nodeRepo.List(ctx, opts)
		recordErr(err)
		if err == nil {
			mu.Lock()
			snapshot.Nodes = nodes
			mu.Unlock()
		}
	}()

	// Deployments
	go func() {
		defer wg.Done()
		deployments, err := s.deploymentRepo.List(ctx, "", opts)
		recordErr(err)
		if err == nil {
			mu.Lock()
			snapshot.Deployments = deployments
			mu.Unlock()
		}
	}()

	// StatefulSets
	go func() {
		defer wg.Done()
		statefulSets, err := s.statefulSetRepo.List(ctx, "", opts)
		recordErr(err)
		if err == nil {
			mu.Lock()
			snapshot.StatefulSets = statefulSets
			mu.Unlock()
		}
	}()

	// DaemonSets
	go func() {
		defer wg.Done()
		daemonSets, err := s.daemonSetRepo.List(ctx, "", opts)
		recordErr(err)
		if err == nil {
			mu.Lock()
			snapshot.DaemonSets = daemonSets
			mu.Unlock()
		}
	}()

	// ReplicaSets
	go func() {
		defer wg.Done()
		replicaSets, err := s.replicaSetRepo.List(ctx, "", opts)
		recordErr(err)
		if err == nil {
			mu.Lock()
			snapshot.ReplicaSets = replicaSets
			mu.Unlock()
		}
	}()

	// Services
	go func() {
		defer wg.Done()
		services, err := s.serviceRepo.List(ctx, "", opts)
		recordErr(err)
		if err == nil {
			mu.Lock()
			snapshot.Services = services
			mu.Unlock()
		}
	}()

	// Ingresses
	go func() {
		defer wg.Done()
		ingresses, err := s.ingressRepo.List(ctx, "", opts)
		recordErr(err)
		if err == nil {
			mu.Lock()
			snapshot.Ingresses = ingresses
			mu.Unlock()
		}
	}()

	// ConfigMaps
	go func() {
		defer wg.Done()
		configMaps, err := s.configMapRepo.List(ctx, "", opts)
		recordErr(err)
		if err == nil {
			mu.Lock()
			snapshot.ConfigMaps = configMaps
			mu.Unlock()
		}
	}()

	// Secrets
	go func() {
		defer wg.Done()
		secrets, err := s.secretRepo.List(ctx, "", opts)
		recordErr(err)
		if err == nil {
			mu.Lock()
			snapshot.Secrets = secrets
			mu.Unlock()
		}
	}()

	// Namespaces
	go func() {
		defer wg.Done()
		namespaces, err := s.namespaceRepo.List(ctx, opts)
		recordErr(err)
		if err == nil {
			mu.Lock()
			snapshot.Namespaces = namespaces
			mu.Unlock()
		}
	}()

	// Events
	go func() {
		defer wg.Done()
		events, err := s.eventRepo.List(ctx, "", opts)
		recordErr(err)
		if err == nil {
			mu.Lock()
			snapshot.Events = events
			mu.Unlock()
		}
	}()

	// Jobs
	go func() {
		defer wg.Done()
		jobs, err := s.jobRepo.List(ctx, "", opts)
		recordErr(err)
		if err == nil {
			mu.Lock()
			snapshot.Jobs = jobs
			mu.Unlock()
		}
	}()

	// CronJobs
	go func() {
		defer wg.Done()
		cronJobs, err := s.cronJobRepo.List(ctx, "", opts)
		recordErr(err)
		if err == nil {
			mu.Lock()
			snapshot.CronJobs = cronJobs
			mu.Unlock()
		}
	}()

	// PersistentVolumes
	go func() {
		defer wg.Done()
		pvs, err := s.pvRepo.List(ctx, opts)
		recordErr(err)
		if err == nil {
			mu.Lock()
			snapshot.PersistentVolumes = pvs
			mu.Unlock()
		}
	}()

	// PersistentVolumeClaims
	go func() {
		defer wg.Done()
		pvcs, err := s.pvcRepo.List(ctx, "", opts)
		recordErr(err)
		if err == nil {
			mu.Lock()
			snapshot.PersistentVolumeClaims = pvcs
			mu.Unlock()
		}
	}()

	// ResourceQuotas
	go func() {
		defer wg.Done()
		rqs, err := s.resourceQuotaRepo.List(ctx, "", opts)
		recordErr(err)
		if err == nil {
			mu.Lock()
			snapshot.ResourceQuotas = rqs
			mu.Unlock()
		}
	}()

	// LimitRanges
	go func() {
		defer wg.Done()
		lrs, err := s.limitRangeRepo.List(ctx, "", opts)
		recordErr(err)
		if err == nil {
			mu.Lock()
			snapshot.LimitRanges = lrs
			mu.Unlock()
		}
	}()

	// NetworkPolicies
	go func() {
		defer wg.Done()
		nps, err := s.networkPolicyRepo.List(ctx, "", opts)
		recordErr(err)
		if err == nil {
			mu.Lock()
			snapshot.NetworkPolicies = nps
			mu.Unlock()
		}
	}()

	// ServiceAccounts
	go func() {
		defer wg.Done()
		sas, err := s.serviceAccountRepo.List(ctx, "", opts)
		recordErr(err)
		if err == nil {
			mu.Lock()
			snapshot.ServiceAccounts = sas
			mu.Unlock()
		}
	}()

	// SLO 数据 (可选，含路由映射)
	// SLO 失败不阻断 K8s 快照上报，只记录警告
	if s.sloRepo != nil {
		go func() {
			defer wg.Done()
			sloData, err := s.sloRepo.Collect(ctx)
			if err != nil {
				// SLO 是可选功能，不传入 recordErr 以免阻断快照
				return
			}
			if sloData != nil {
				mu.Lock()
				snapshot.SLOData = sloData
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	// 聚合节点指标
	if s.metricsRepo != nil {
		snapshot.NodeMetrics = s.metricsRepo.GetAll()
	}

	// 统计每个 Namespace 的资源数量
	s.calculateNamespaceResources(snapshot)

	// 生成摘要
	snapshot.Summary = s.generateSummary(snapshot)

	return snapshot, firstErr
}

// calculateNamespaceResources 统计每个 Namespace 的资源数量
//
// 遍历快照中的所有资源，按 Namespace 分组统计，
// 然后更新 Namespaces 列表中每个 Namespace 的 Resources 字段。
// 同时关联 ResourceQuotas 和 LimitRanges 到对应的 Namespace。
func (s *snapshotService) calculateNamespaceResources(snapshot *model_v2.ClusterSnapshot) {
	// 初始化统计 map
	type nsStats struct {
		pods, podsRunning, podsPending, podsFailed, podsSucceeded int
		deployments, statefulSets, daemonSets, replicaSets        int
		jobs, cronJobs                                             int
		services, ingresses, networkPolicies                       int
		configMaps, secrets, serviceAccounts                       int
		pvcs                                                       int
		quotas                                                     []model_v2.ResourceQuota
		limitRanges                                                []model_v2.LimitRange
	}
	statsByNs := make(map[string]*nsStats)

	// 确保所有 Namespace 都有初始化的统计
	for _, ns := range snapshot.Namespaces {
		statsByNs[ns.GetName()] = &nsStats{}
	}

	// 统计 Pods
	for _, pod := range snapshot.Pods {
		ns := pod.GetNamespace()
		if stats, ok := statsByNs[ns]; ok {
			stats.pods++
			switch pod.Status.Phase {
			case "Running":
				stats.podsRunning++
			case "Pending":
				stats.podsPending++
			case "Failed":
				stats.podsFailed++
			case "Succeeded":
				stats.podsSucceeded++
			}
		}
	}

	// 统计 Deployments
	for _, d := range snapshot.Deployments {
		if stats, ok := statsByNs[d.GetNamespace()]; ok {
			stats.deployments++
		}
	}

	// 统计 StatefulSets
	for _, ss := range snapshot.StatefulSets {
		if stats, ok := statsByNs[ss.GetNamespace()]; ok {
			stats.statefulSets++
		}
	}

	// 统计 DaemonSets
	for _, ds := range snapshot.DaemonSets {
		if stats, ok := statsByNs[ds.GetNamespace()]; ok {
			stats.daemonSets++
		}
	}

	// 统计 ReplicaSets
	for _, rs := range snapshot.ReplicaSets {
		if stats, ok := statsByNs[rs.Namespace]; ok {
			stats.replicaSets++
		}
	}

	// 统计 Jobs
	for _, j := range snapshot.Jobs {
		if stats, ok := statsByNs[j.Namespace]; ok {
			stats.jobs++
		}
	}

	// 统计 CronJobs
	for _, cj := range snapshot.CronJobs {
		if stats, ok := statsByNs[cj.Namespace]; ok {
			stats.cronJobs++
		}
	}

	// 统计 Services
	for _, svc := range snapshot.Services {
		if stats, ok := statsByNs[svc.GetNamespace()]; ok {
			stats.services++
		}
	}

	// 统计 Ingresses
	for _, ing := range snapshot.Ingresses {
		if stats, ok := statsByNs[ing.GetNamespace()]; ok {
			stats.ingresses++
		}
	}

	// 统计 NetworkPolicies
	for _, np := range snapshot.NetworkPolicies {
		if stats, ok := statsByNs[np.Namespace]; ok {
			stats.networkPolicies++
		}
	}

	// 统计 ConfigMaps
	for _, cm := range snapshot.ConfigMaps {
		if stats, ok := statsByNs[cm.Namespace]; ok {
			stats.configMaps++
		}
	}

	// 统计 Secrets
	for _, sec := range snapshot.Secrets {
		if stats, ok := statsByNs[sec.Namespace]; ok {
			stats.secrets++
		}
	}

	// 统计 ServiceAccounts
	for _, sa := range snapshot.ServiceAccounts {
		if stats, ok := statsByNs[sa.Namespace]; ok {
			stats.serviceAccounts++
		}
	}

	// 统计 PVCs
	for _, pvc := range snapshot.PersistentVolumeClaims {
		if stats, ok := statsByNs[pvc.Namespace]; ok {
			stats.pvcs++
		}
	}

	// 关联 ResourceQuotas
	for _, rq := range snapshot.ResourceQuotas {
		if stats, ok := statsByNs[rq.Namespace]; ok {
			stats.quotas = append(stats.quotas, rq)
		}
	}

	// 关联 LimitRanges
	for _, lr := range snapshot.LimitRanges {
		if stats, ok := statsByNs[lr.Namespace]; ok {
			stats.limitRanges = append(stats.limitRanges, lr)
		}
	}

	// 更新 Namespace 的 Resources 字段
	for i := range snapshot.Namespaces {
		nsName := snapshot.Namespaces[i].GetName()
		if stats, ok := statsByNs[nsName]; ok {
			snapshot.Namespaces[i].Resources = model_v2.NamespaceResources{
				Pods:            stats.pods,
				PodsRunning:     stats.podsRunning,
				PodsPending:     stats.podsPending,
				PodsFailed:      stats.podsFailed,
				PodsSucceeded:   stats.podsSucceeded,
				Deployments:     stats.deployments,
				StatefulSets:    stats.statefulSets,
				DaemonSets:      stats.daemonSets,
				ReplicaSets:     stats.replicaSets,
				Jobs:            stats.jobs,
				CronJobs:        stats.cronJobs,
				Services:        stats.services,
				Ingresses:       stats.ingresses,
				NetworkPolicies: stats.networkPolicies,
				ConfigMaps:      stats.configMaps,
				Secrets:         stats.secrets,
				ServiceAccounts: stats.serviceAccounts,
				PVCs:            stats.pvcs,
			}
			snapshot.Namespaces[i].Quotas = stats.quotas
			snapshot.Namespaces[i].LimitRanges = stats.limitRanges
		}
	}
}

// generateSummary 生成快照摘要
//
// 遍历快照中的资源，统计各类指标:
//   - Node: 总数、Ready 数
//   - Pod: 总数、Running/Pending/Failed 数
//   - Deployment: 总数、健康 (ReadyReplicas == Replicas) 数
//   - Event: 总数、Warning 类型数
//   - Namespace: 总数
//
// 摘要用于仪表盘快速展示，避免前端遍历完整数据。
func (s *snapshotService) generateSummary(snapshot *model_v2.ClusterSnapshot) model_v2.ClusterSummary {
	summary := model_v2.ClusterSummary{
		TotalNodes:        len(snapshot.Nodes),
		TotalPods:         len(snapshot.Pods),
		TotalDeployments:  len(snapshot.Deployments),
		TotalStatefulSets: len(snapshot.StatefulSets),
		TotalDaemonSets:   len(snapshot.DaemonSets),
		TotalServices:     len(snapshot.Services),
		TotalIngresses:    len(snapshot.Ingresses),
		TotalNamespaces:   len(snapshot.Namespaces),
		TotalEvents:       len(snapshot.Events),
	}

	// 统计 Node 状态
	for _, node := range snapshot.Nodes {
		if node.IsReady() {
			summary.ReadyNodes++
		}
	}

	// 统计 Pod 状态
	for _, pod := range snapshot.Pods {
		switch pod.Status.Phase {
		case "Running":
			summary.RunningPods++
		case "Pending":
			summary.PendingPods++
		case "Failed":
			summary.FailedPods++
		}
	}

	// 统计 Deployment 状态
	for _, deploy := range snapshot.Deployments {
		if deploy.IsHealthy() {
			summary.HealthyDeployments++
		}
	}

	// 统计 Event 状态
	for _, event := range snapshot.Events {
		if event.IsWarning() {
			summary.WarningEvents++
		}
	}

	return summary
}

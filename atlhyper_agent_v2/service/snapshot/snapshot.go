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
//
// 文件分布:
//   - snapshot.go       — 结构体定义、K8s 采集、Namespace 统计
//   - otel_collector.go — OTel 快照缓存与聚合
//   - slo_collector.go  — SLO 多窗口数据采集
package snapshot

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"AtlHyper/atlhyper_agent_v2/concentrator"
	"AtlHyper/atlhyper_agent_v2/model"
	"AtlHyper/atlhyper_agent_v2/repository"
	"AtlHyper/atlhyper_agent_v2/service"
	"AtlHyper/common/logger"
	"AtlHyper/model_v3/cluster"
	"AtlHyper/model_v3/slo"
)

var log = logger.Module("Snapshot")

// snapshotService 快照采集服务实现
//
// 依赖 20 个 K8s Repository + 可选的 OTel 概览仓库。
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
	// 路由映射: serviceKey → 对外域名（供 SLO displayName 使用）
	routeRepo repository.RouteRepository

	// 上一轮产出的快照：某类资源本轮采集失败时沿用其中的同类数据。
	// 只在 Collect 所在的单个 goroutine 里读写。
	lastSnapshot *cluster.ClusterSnapshot

	// OTel 仓库 (可选，从 ClickHouse 聚合)
	otelSummaryRepo repository.OTelSummaryRepository
	dashboardRepo   repository.OTelDashboardRepository

	// OTel 缓存（分离 TTL：Summary 慢变化 5min / Dashboard 列表快变化 30s）
	otelCache              *cluster.OTelSnapshot
	otelCacheTime          time.Time
	otelDashboardCache     *dashboardCacheData
	otelDashboardCacheTime time.Time

	// SLO 多窗口缓存（TTL 按窗口大小递增）
	sloWindowCaches map[string]*sloWindowCache

	// Concentrator 预聚合时序（可选）
	conc concentrator.TimeSeriesAggregator
}

// sloWindowCache SLO 窗口数据缓存
type sloWindowCache struct {
	data      *slo.SLOWindowData
	fetchedAt time.Time
}

// dashboardCacheData Dashboard 列表数据缓存
type dashboardCacheData struct {
	snapshot *cluster.OTelSnapshot // 仅 Dashboard 列表字段
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
	routeRepo repository.RouteRepository,
	otelSummaryRepo repository.OTelSummaryRepository,
	dashboardRepo repository.OTelDashboardRepository,
	conc concentrator.TimeSeriesAggregator,
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
		routeRepo:          routeRepo,
		otelSummaryRepo:    otelSummaryRepo,
		dashboardRepo:      dashboardRepo,
		conc:               conc,
	}
}

// Collect 采集集群快照
//
// 并发采集 20 种 K8s 资源，组装为完整的 ClusterSnapshot。
// 采集完成后生成统计摘要，用于仪表盘快速展示。
//
// 部分失败的处理（某一类资源 List 超时、无权限等）:
//   - 有上一轮快照：失败的那类沿用上一轮的数据，快照照常产出
//   - 没有上一轮（启动后第一轮）：Pod / Node 失败则返回错误，不产出快照；
//     其他资源留空照常产出
//
// 整份快照因一类资源失败而作废，会让全部资源一起停止更新，代价远大于某一类数据旧一轮。
func (s *snapshotService) Collect(ctx context.Context) (*cluster.ClusterSnapshot, error) {
	snapshot := &cluster.ClusterSnapshot{
		ClusterID: s.clusterID,
		FetchedAt: time.Now(),
	}

	collectors := s.kindCollectors(ctx, snapshot)
	failed := runCollectors(collectors)

	if len(failed) > 0 {
		var reused, empty []string
		for _, collector := range collectors {
			err, isFailed := failed[collector.kind]
			if !isFailed {
				continue
			}
			switch {
			case s.lastSnapshot != nil:
				collector.reuse(s.lastSnapshot)
				reused = append(reused, collector.kind)
			case collector.essential:
				return nil, fmt.Errorf("collect %s: %w", collector.kind, err)
			default:
				empty = append(empty, collector.kind)
			}
		}
		for kind, err := range failed {
			log.Warn("资源采集失败", "kind", kind, "err", err)
		}
		log.Warn("本轮快照含未更新的资源",
			"沿用上一轮", strings.Join(sortedKinds(reused), ","),
			"留空", strings.Join(sortedKinds(empty), ","))
	}
	s.lastSnapshot = snapshot

	// 补全 Node 的 Pod 使用量统计（需要 Nodes + Pods 都就绪）
	enrichNodePodCount(snapshot)

	// 聚合 OTel 快照（带缓存）
	if s.otelSummaryRepo != nil || s.dashboardRepo != nil {
		otelSnapshot := s.getOTelSnapshot(ctx)
		if otelSnapshot != nil {
			snapshot.OTel = otelSnapshot
		}
	}

	// 统计每个 Namespace 的资源数量
	s.calculateNamespaceResources(snapshot)

	// 生成摘要
	snapshot.Summary = snapshot.GenerateSummary()

	return snapshot, nil
}

// kindCollectors 20 类 K8s 资源的采集器，结果写入 snapshot
func (s *snapshotService) kindCollectors(ctx context.Context, snapshot *cluster.ClusterSnapshot) []kindCollector {
	opts := model.ListOptions{}
	type snap = cluster.ClusterSnapshot
	return []kindCollector{
		collectorFor("pods", true, snapshot,
			func(c *snap) *[]cluster.Pod { return &c.Pods },
			func() ([]cluster.Pod, error) { return s.podRepo.List(ctx, "", opts) }),
		collectorFor("nodes", true, snapshot,
			func(c *snap) *[]cluster.Node { return &c.Nodes },
			func() ([]cluster.Node, error) { return s.nodeRepo.List(ctx, opts) }),
		collectorFor("deployments", false, snapshot,
			func(c *snap) *[]cluster.Deployment { return &c.Deployments },
			func() ([]cluster.Deployment, error) { return s.deploymentRepo.List(ctx, "", opts) }),
		collectorFor("statefulsets", false, snapshot,
			func(c *snap) *[]cluster.StatefulSet { return &c.StatefulSets },
			func() ([]cluster.StatefulSet, error) { return s.statefulSetRepo.List(ctx, "", opts) }),
		collectorFor("daemonsets", false, snapshot,
			func(c *snap) *[]cluster.DaemonSet { return &c.DaemonSets },
			func() ([]cluster.DaemonSet, error) { return s.daemonSetRepo.List(ctx, "", opts) }),
		collectorFor("replicasets", false, snapshot,
			func(c *snap) *[]cluster.ReplicaSet { return &c.ReplicaSets },
			func() ([]cluster.ReplicaSet, error) { return s.replicaSetRepo.List(ctx, "", opts) }),
		collectorFor("services", false, snapshot,
			func(c *snap) *[]cluster.Service { return &c.Services },
			func() ([]cluster.Service, error) { return s.serviceRepo.List(ctx, "", opts) }),
		collectorFor("ingresses", false, snapshot,
			func(c *snap) *[]cluster.Ingress { return &c.Ingresses },
			func() ([]cluster.Ingress, error) { return s.ingressRepo.List(ctx, "", opts) }),
		collectorFor("configmaps", false, snapshot,
			func(c *snap) *[]cluster.ConfigMap { return &c.ConfigMaps },
			func() ([]cluster.ConfigMap, error) { return s.configMapRepo.List(ctx, "", opts) }),
		collectorFor("secrets", false, snapshot,
			func(c *snap) *[]cluster.Secret { return &c.Secrets },
			func() ([]cluster.Secret, error) { return s.secretRepo.List(ctx, "", opts) }),
		collectorFor("namespaces", false, snapshot,
			func(c *snap) *[]cluster.Namespace { return &c.Namespaces },
			func() ([]cluster.Namespace, error) { return s.namespaceRepo.List(ctx, opts) }),
		collectorFor("events", false, snapshot,
			func(c *snap) *[]cluster.Event { return &c.Events },
			func() ([]cluster.Event, error) { return s.eventRepo.List(ctx, "", opts) }),
		collectorFor("jobs", false, snapshot,
			func(c *snap) *[]cluster.Job { return &c.Jobs },
			func() ([]cluster.Job, error) { return s.jobRepo.List(ctx, "", opts) }),
		collectorFor("cronjobs", false, snapshot,
			func(c *snap) *[]cluster.CronJob { return &c.CronJobs },
			func() ([]cluster.CronJob, error) { return s.cronJobRepo.List(ctx, "", opts) }),
		collectorFor("persistentvolumes", false, snapshot,
			func(c *snap) *[]cluster.PersistentVolume { return &c.PersistentVolumes },
			func() ([]cluster.PersistentVolume, error) { return s.pvRepo.List(ctx, opts) }),
		collectorFor("persistentvolumeclaims", false, snapshot,
			func(c *snap) *[]cluster.PersistentVolumeClaim { return &c.PersistentVolumeClaims },
			func() ([]cluster.PersistentVolumeClaim, error) { return s.pvcRepo.List(ctx, "", opts) }),
		collectorFor("resourcequotas", false, snapshot,
			func(c *snap) *[]cluster.ResourceQuota { return &c.ResourceQuotas },
			func() ([]cluster.ResourceQuota, error) { return s.resourceQuotaRepo.List(ctx, "", opts) }),
		collectorFor("limitranges", false, snapshot,
			func(c *snap) *[]cluster.LimitRange { return &c.LimitRanges },
			func() ([]cluster.LimitRange, error) { return s.limitRangeRepo.List(ctx, "", opts) }),
		collectorFor("networkpolicies", false, snapshot,
			func(c *snap) *[]cluster.NetworkPolicy { return &c.NetworkPolicies },
			func() ([]cluster.NetworkPolicy, error) { return s.networkPolicyRepo.List(ctx, "", opts) }),
		collectorFor("serviceaccounts", false, snapshot,
			func(c *snap) *[]cluster.ServiceAccount { return &c.ServiceAccounts },
			func() ([]cluster.ServiceAccount, error) { return s.serviceAccountRepo.List(ctx, "", opts) }),
	}
}

// calculateNamespaceResources 统计每个 Namespace 的资源数量
//
// 遍历快照中的所有资源，按 Namespace 分组统计，
// 然后更新 Namespaces 列表中每个 Namespace 的 Resources 字段。
// 同时关联 ResourceQuotas 和 LimitRanges 到对应的 Namespace。
func (s *snapshotService) calculateNamespaceResources(snapshot *cluster.ClusterSnapshot) {
	// 初始化统计 map
	type nsStats struct {
		pods, podsRunning, podsPending, podsFailed, podsSucceeded int
		deployments, statefulSets, daemonSets, replicaSets        int
		jobs, cronJobs                                            int
		services, ingresses, networkPolicies                      int
		configMaps, secrets, serviceAccounts                      int
		pvcs                                                      int
		quotas                                                    []cluster.ResourceQuota
		limitRanges                                               []cluster.LimitRange
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
			snapshot.Namespaces[i].Resources = cluster.NamespaceResources{
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

// enrichNodePodCount 补全每个 Node 的 Pod 使用量统计
// 需在 Nodes 和 Pods 都采集完成后调用
func enrichNodePodCount(snapshot *cluster.ClusterSnapshot) {
	// 统计每个 Node 上的 Pod 数量
	podCountByNode := make(map[string]int, len(snapshot.Nodes))
	for i := range snapshot.Pods {
		nodeName := snapshot.Pods[i].Summary.NodeName
		if nodeName != "" {
			podCountByNode[nodeName]++
		}
	}

	for i := range snapshot.Nodes {
		node := &snapshot.Nodes[i]
		if node.Metrics == nil {
			continue
		}

		podCount := podCountByNode[node.GetName()]
		podsCapacity := 0
		if v := node.Allocatable.Pods; v != "" {
			podsCapacity, _ = strconv.Atoi(v)
		}
		if podsCapacity == 0 {
			if v := node.Capacity.Pods; v != "" {
				podsCapacity, _ = strconv.Atoi(v)
			}
		}

		var utilPct float64
		if podsCapacity > 0 {
			utilPct = math.Round(float64(podCount)/float64(podsCapacity)*10000) / 100
		}
		node.Metrics.Pods = cluster.PodCountMetric{
			Used:     podCount,
			Capacity: podsCapacity,
			UtilPct:  utilPct,
		}
	}
}

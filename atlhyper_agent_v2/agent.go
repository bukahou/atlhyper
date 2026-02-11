// Package atlhyper_agent_v2 Agent V2 核心包
//
// 本包提供 Agent 的启动器，负责:
//   - 初始化所有依赖 (SDK, Gateway, Repository, Service, Scheduler)
//   - 依赖注入和组装
//   - 生命周期管理 (启动、运行、停止)
//
// 使用方式:
//
//	config.LoadConfig()
//	agent, err := atlhyper_agent_v2.New()
//	agent.Run(ctx)
package atlhyper_agent_v2

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"AtlHyper/atlhyper_agent_v2/config"
	"AtlHyper/atlhyper_agent_v2/gateway"
	"AtlHyper/atlhyper_agent_v2/sdk/impl/receiver"
	"AtlHyper/atlhyper_agent_v2/repository"
	k8srepo "AtlHyper/atlhyper_agent_v2/repository/k8s"
	metricsrepo "AtlHyper/atlhyper_agent_v2/repository/metrics"
	slorepo "AtlHyper/atlhyper_agent_v2/repository/slo"
	"AtlHyper/atlhyper_agent_v2/scheduler"
	sdkpkg "AtlHyper/atlhyper_agent_v2/sdk"
	"AtlHyper/atlhyper_agent_v2/sdk/impl/ingress"
	k8spkg "AtlHyper/atlhyper_agent_v2/sdk/impl/k8s"
	otelpkg "AtlHyper/atlhyper_agent_v2/sdk/impl/otel"
	commandsvc "AtlHyper/atlhyper_agent_v2/service/command"
	snapshotsvc "AtlHyper/atlhyper_agent_v2/service/snapshot"
	"AtlHyper/common/logger"
)

var log = logger.Module("Agent")

// Agent 是 Agent V2 的主结构体
// 封装调度器，提供启动/运行/停止接口
type Agent struct {
	scheduler   *scheduler.Scheduler
	receiverSvr sdkpkg.ReceiverClient
}

// New 创建并初始化 Agent 实例
//
// 使用 config.GlobalConfig 中的配置初始化各层组件。
// 调用前必须先调用 config.LoadConfig()。
//
// 初始化顺序:
//  1. SDK 层 - 连接 K8s API Server
//  2. Gateway 层 - 创建 Master 通信客户端
//  3. Repository 层 - 创建各资源仓库
//  4. Service 层 - 创建业务服务
//  5. Scheduler 层 - 创建调度器
//
// 返回:
//   - *Agent: Agent 实例
//   - error: 初始化错误 (通常是 K8s 连接失败)
func New() (*Agent, error) {
	cfg := &config.GlobalConfig

	// 1. 初始化 SDK (连接 K8s)
	k8sClient, err := k8spkg.NewClient(cfg.Kubernetes.KubeConfig)
	if err != nil {
		return nil, err
	}
	log.Info("K8s 客户端初始化完成")

	// 如果未配置 ClusterID，自动获取集群 UID (kube-system namespace 的 UID)
	if cfg.Agent.ClusterID == "" {
		ns, err := k8sClient.GetNamespace(context.Background(), "kube-system")
		if err != nil {
			return nil, fmt.Errorf("failed to get cluster UID: %w", err)
		}
		cfg.Agent.ClusterID = string(ns.UID)
		log.Info("自动检测 ClusterID", "cluster", cfg.Agent.ClusterID)
	}

	// 2. 初始化 Gateway (Master 通信)
	masterGw := gateway.NewMasterGateway(cfg.Master.URL, cfg.Agent.ClusterID, cfg.Timeout.HTTPClient)

	// 3. 初始化 Repository (数据访问层)
	repos := initRepositories(k8sClient)

	// 3.1 初始化 OTel 客户端（SLO + 节点指标共用）
	var otelClient sdkpkg.OTelClient
	if cfg.SLO.Enabled {
		otelClient = otelpkg.NewOTelClient(cfg.SLO.OTelMetricsURL, cfg.SLO.OTelHealthURL, cfg.SLO.ScrapeTimeout)
		log.Info("OTel 客户端初始化完成", "url", cfg.SLO.OTelMetricsURL)
	}

	// 3.2 初始化 MetricsRepository (节点指标)
	// 优先用 OTel 拉取 node_exporter 指标，降级到 Receiver 被动接收
	var metricsRepo repository.MetricsRepository
	var receiverClient sdkpkg.ReceiverClient
	if cfg.MetricsSDK.Enabled {
		receiverClient = receiver.NewServer(cfg.MetricsSDK.Port)
	}
	if otelClient != nil {
		metricsRepo = metricsrepo.NewMetricsRepository(otelClient, receiverClient)
		log.Info("MetricsRepository 初始化完成 (OTel 模式)")
	} else if receiverClient != nil {
		metricsRepo = metricsrepo.NewLegacyMetricsRepository(receiverClient)
		log.Info("MetricsRepository 初始化完成 (Receiver 降级模式)")
	}

	// 3.3 初始化 SLORepository (可选)
	var sloRepo repository.SLORepository
	if cfg.SLO.Enabled && otelClient != nil {
		ingressClient := ingress.NewIngressClient(k8sClient)
		excludeNS := cfg.SLO.ExcludeNamespaces
		if len(excludeNS) == 0 {
			excludeNS = []string{"linkerd", "linkerd-viz", "kube-system", "otel"}
		}
		sloRepo = slorepo.NewSLORepository(otelClient, ingressClient, excludeNS)
		log.Info("SLO Repository 初始化完成")
	}

	// 4. 初始化 Service (业务逻辑层)
	snapshotSvc := snapshotsvc.NewSnapshotService(
		cfg.Agent.ClusterID,
		repos.pod, repos.node, repos.deployment,
		repos.statefulSet, repos.daemonSet, repos.replicaSet,
		repos.service, repos.ingress, repos.configMap,
		repos.secret, repos.namespace, repos.event,
		repos.job, repos.cronJob, repos.pv, repos.pvc,
		repos.resourceQuota, repos.limitRange,
		repos.networkPolicy, repos.serviceAccount,
		metricsRepo,
		sloRepo,
	)

	commandSvc := commandsvc.NewCommandService(repos.pod, repos.generic)

	// 5. 初始化 Scheduler (调度层)
	schedCfg := scheduler.Config{
		SnapshotInterval:    cfg.Scheduler.SnapshotInterval,
		CommandPollInterval: cfg.Scheduler.CommandPollInterval,
		HeartbeatInterval:   cfg.Scheduler.HeartbeatInterval,
		SnapshotTimeout:     cfg.Timeout.SnapshotCollect,
		CommandPollTimeout:  cfg.Timeout.CommandPoll,
		HeartbeatTimeout:    cfg.Timeout.Heartbeat,
	}
	sched := scheduler.New(schedCfg, snapshotSvc, commandSvc, masterGw, metricsRepo)

	return &Agent{
		scheduler:   sched,
		receiverSvr: receiverClient,
	}, nil
}

// Run 运行 Agent
//
// 启动调度器后阻塞等待退出信号 (SIGINT/SIGTERM)。
// 收到信号后优雅停止调度器。
//
// 参数:
//   - ctx: 上下文，可用于外部取消
//
// 返回:
//   - error: 调度器停止时的错误
func (a *Agent) Run(ctx context.Context) error {
	// 启动 Receiver SDK (如果启用)
	if a.receiverSvr != nil {
		if err := a.receiverSvr.Start(); err != nil {
			return err
		}
	}

	// 启动调度器 (开始快照采集、指令轮询、心跳)
	if err := a.scheduler.Start(ctx); err != nil {
		return err
	}

	log.Info("Agent 启动成功")

	// 等待退出信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	// 优雅停止
	log.Info("正在关闭...")

	// 停止 Receiver SDK
	if a.receiverSvr != nil {
		if err := a.receiverSvr.Stop(); err != nil {
			log.Error("停止 Metrics SDK 失败", "err", err)
		}
	}

	return a.scheduler.Stop()
}

// =============================================================================
// 内部辅助
// =============================================================================

// repositories 聚合所有 Repository 实例
// 用于简化依赖传递
type repositories struct {
	pod            repository.PodRepository
	node           repository.NodeRepository
	deployment     repository.DeploymentRepository
	statefulSet    repository.StatefulSetRepository
	daemonSet      repository.DaemonSetRepository
	replicaSet     repository.ReplicaSetRepository
	service        repository.ServiceRepository
	ingress        repository.IngressRepository
	configMap      repository.ConfigMapRepository
	secret         repository.SecretRepository
	namespace      repository.NamespaceRepository
	event          repository.EventRepository
	job            repository.JobRepository
	cronJob        repository.CronJobRepository
	pv             repository.PersistentVolumeRepository
	pvc            repository.PersistentVolumeClaimRepository
	resourceQuota  repository.ResourceQuotaRepository
	limitRange     repository.LimitRangeRepository
	networkPolicy  repository.NetworkPolicyRepository
	serviceAccount repository.ServiceAccountRepository
	generic        repository.GenericRepository
}

// initRepositories 初始化所有 Repository
// 每个 Repository 封装对应 K8s 资源的 CRUD 操作
func initRepositories(client sdkpkg.K8sClient) *repositories {
	return &repositories{
		pod:            k8srepo.NewPodRepository(client),
		node:           k8srepo.NewNodeRepository(client),
		deployment:     k8srepo.NewDeploymentRepository(client),
		statefulSet:    k8srepo.NewStatefulSetRepository(client),
		daemonSet:      k8srepo.NewDaemonSetRepository(client),
		replicaSet:     k8srepo.NewReplicaSetRepository(client),
		service:        k8srepo.NewServiceRepository(client),
		ingress:        k8srepo.NewIngressRepository(client),
		configMap:      k8srepo.NewConfigMapRepository(client),
		secret:         k8srepo.NewSecretRepository(client),
		namespace:      k8srepo.NewNamespaceRepository(client),
		event:          k8srepo.NewEventRepository(client),
		job:            k8srepo.NewJobRepository(client),
		cronJob:        k8srepo.NewCronJobRepository(client),
		pv:             k8srepo.NewPersistentVolumeRepository(client),
		pvc:            k8srepo.NewPersistentVolumeClaimRepository(client),
		resourceQuota:  k8srepo.NewResourceQuotaRepository(client),
		limitRange:     k8srepo.NewLimitRangeRepository(client),
		networkPolicy:  k8srepo.NewNetworkPolicyRepository(client),
		serviceAccount: k8srepo.NewServiceAccountRepository(client),
		generic:        k8srepo.NewGenericRepository(client),
	}
}

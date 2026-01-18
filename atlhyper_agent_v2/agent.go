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
	"log"
	"os"
	"os/signal"
	"syscall"

	"AtlHyper/atlhyper_agent_v2/config"
	"AtlHyper/atlhyper_agent_v2/gateway"
	"AtlHyper/atlhyper_agent_v2/repository"
	"AtlHyper/atlhyper_agent_v2/scheduler"
	"AtlHyper/atlhyper_agent_v2/sdk"
	"AtlHyper/atlhyper_agent_v2/sdk/impl"
	"AtlHyper/atlhyper_agent_v2/service"
)

// Agent 是 Agent V2 的主结构体
// 封装调度器，提供启动/运行/停止接口
type Agent struct {
	scheduler *scheduler.Scheduler
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
	k8sClient, err := impl.NewClient(cfg.Kubernetes.KubeConfig)
	if err != nil {
		return nil, err
	}
	log.Println("[Agent] K8s 客户端初始化完成")

	// 如果未配置 ClusterID，自动获取集群 UID (kube-system namespace 的 UID)
	if cfg.Agent.ClusterID == "" {
		ns, err := k8sClient.GetNamespace(context.Background(), "kube-system")
		if err != nil {
			return nil, fmt.Errorf("failed to get cluster UID: %w", err)
		}
		cfg.Agent.ClusterID = string(ns.UID)
		log.Printf("[Agent] 自动检测 ClusterID: %s", cfg.Agent.ClusterID)
	}

	// 2. 初始化 Gateway (Master 通信)
	masterGw := gateway.NewMasterGateway(cfg.Master.URL, cfg.Agent.ClusterID, cfg.Timeout.HTTPClient)

	// 3. 初始化 Repository (数据访问层)
	repos := initRepositories(k8sClient)

	// 4. 初始化 Service (业务逻辑层)
	snapshotSvc := service.NewSnapshotService(
		cfg.Agent.ClusterID,
		repos.pod, repos.node, repos.deployment,
		repos.statefulSet, repos.daemonSet, repos.replicaSet,
		repos.service, repos.ingress, repos.configMap,
		repos.secret, repos.namespace, repos.event,
		repos.job, repos.cronJob, repos.pv, repos.pvc,
		repos.resourceQuota, repos.limitRange,
		repos.networkPolicy, repos.serviceAccount,
	)

	commandSvc := service.NewCommandService(repos.pod, repos.generic)

	// 5. 初始化 Scheduler (调度层)
	schedCfg := scheduler.Config{
		SnapshotInterval:    cfg.Scheduler.SnapshotInterval,
		CommandPollInterval: cfg.Scheduler.CommandPollInterval,
		HeartbeatInterval:   cfg.Scheduler.HeartbeatInterval,
		SnapshotTimeout:     cfg.Timeout.SnapshotCollect,
		CommandPollTimeout:  cfg.Timeout.CommandPoll,
		HeartbeatTimeout:    cfg.Timeout.Heartbeat,
	}
	sched := scheduler.New(schedCfg, snapshotSvc, commandSvc, masterGw)

	return &Agent{
		scheduler: sched,
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
	// 启动调度器 (开始快照采集、指令轮询、心跳)
	if err := a.scheduler.Start(ctx); err != nil {
		return err
	}

	log.Println("[Agent] Agent 启动成功")

	// 等待退出信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	// 优雅停止
	log.Println("[Agent] 正在关闭...")
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
func initRepositories(client sdk.K8sClient) *repositories {
	return &repositories{
		pod:            repository.NewPodRepository(client),
		node:           repository.NewNodeRepository(client),
		deployment:     repository.NewDeploymentRepository(client),
		statefulSet:    repository.NewStatefulSetRepository(client),
		daemonSet:      repository.NewDaemonSetRepository(client),
		replicaSet:     repository.NewReplicaSetRepository(client),
		service:        repository.NewServiceRepository(client),
		ingress:        repository.NewIngressRepository(client),
		configMap:      repository.NewConfigMapRepository(client),
		secret:         repository.NewSecretRepository(client),
		namespace:      repository.NewNamespaceRepository(client),
		event:          repository.NewEventRepository(client),
		job:            repository.NewJobRepository(client),
		cronJob:        repository.NewCronJobRepository(client),
		pv:             repository.NewPersistentVolumeRepository(client),
		pvc:            repository.NewPersistentVolumeClaimRepository(client),
		resourceQuota:  repository.NewResourceQuotaRepository(client),
		limitRange:     repository.NewLimitRangeRepository(client),
		networkPolicy:  repository.NewNetworkPolicyRepository(client),
		serviceAccount: repository.NewServiceAccountRepository(client),
		generic:        repository.NewGenericRepository(client),
	}
}

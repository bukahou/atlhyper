package scheduler

import (
	"context"
	"sync"
	"time"

	"AtlHyper/atlhyper_agent_v2/gateway"
	"AtlHyper/atlhyper_agent_v2/service"
	"AtlHyper/common/logger"
)

var log = logger.Module("Scheduler")

// Config 调度器配置
type Config struct {
	// SnapshotInterval 快照采集间隔
	// 每隔此时间采集一次集群资源并推送给 Master
	SnapshotInterval time.Duration

	// CommandPollInterval 指令轮询间隔
	// 长轮询返回后等待此时间再次轮询
	CommandPollInterval time.Duration

	// HeartbeatInterval 心跳间隔
	// 每隔此时间向 Master 发送心跳
	HeartbeatInterval time.Duration

	// SnapshotTimeout 快照采集操作超时
	SnapshotTimeout time.Duration

	// CommandPollTimeout 指令轮询操作超时
	CommandPollTimeout time.Duration

	// HeartbeatTimeout 心跳操作超时
	HeartbeatTimeout time.Duration

	// SLO 配置
	SLOEnabled       bool          // 是否启用 SLO 采集
	SLOScrapeInterval time.Duration // SLO 采集间隔
	SLOScrapeTimeout  time.Duration // SLO 采集超时
}

// Scheduler 调度器
//
// 管理 Agent 各项后台任务的生命周期:
//   - 快照采集循环 - 定时采集集群资源，推送给 Master
//   - 指令轮询循环 - 长轮询获取 Master 指令，执行后上报结果
//   - 心跳循环 - 定时向 Master 发送心跳，维持连接状态
//   - SLO 采集循环 - 定时采集 Ingress 指标，推送给 Master
type Scheduler struct {
	config Config

	// 依赖的服务
	snapshotSvc service.SnapshotService
	commandSvc  service.CommandService
	sloSvc      service.SLOService
	masterGw    gateway.MasterGateway

	// 生命周期控制
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// 状态追踪（用于变化检测）
	lastPodCount        int
	lastNodeCount       int
	lastDeploymentCount int
}

// New 创建调度器
//
// 参数:
//   - config: 调度器配置
//   - snapshotSvc: 快照服务，用于采集集群资源
//   - commandSvc: 指令服务，用于执行 Master 下发的指令
//   - sloSvc: SLO 服务，用于采集 Ingress 指标（可为 nil）
//   - masterGw: Master 网关，用于与 Master 通信
func New(
	config Config,
	snapshotSvc service.SnapshotService,
	commandSvc service.CommandService,
	sloSvc service.SLOService,
	masterGw gateway.MasterGateway,
) *Scheduler {
	return &Scheduler{
		config:      config,
		snapshotSvc: snapshotSvc,
		commandSvc:  commandSvc,
		sloSvc:      sloSvc,
		masterGw:    masterGw,
	}
}

// Start 启动调度器
func (s *Scheduler) Start(ctx context.Context) error {
	s.ctx, s.cancel = context.WithCancel(ctx)

	// 计算后台任务数量
	taskCount := 4 // 快照 + ops轮询 + ai轮询 + 心跳
	if s.config.SLOEnabled && s.sloSvc != nil {
		taskCount++ // SLO 采集
	}

	// 启动后台任务
	s.wg.Add(taskCount)
	go s.runSnapshotLoop()     // 快照采集
	go s.runCommandLoop("ops") // 系统操作指令轮询
	go s.runCommandLoop("ai")  // AI 查询指令轮询
	go s.runHeartbeatLoop()    // 心跳

	// SLO 采集（可选）
	if s.config.SLOEnabled && s.sloSvc != nil {
		go s.runSLOLoop()
		log.Info("SLO 采集已启用", "interval", s.config.SLOScrapeInterval)
	}

	log.Info("调度器已启动")
	return nil
}

// Stop 停止调度器
func (s *Scheduler) Stop() error {
	if s.cancel != nil {
		s.cancel() // 通知所有 goroutine 停止
	}
	s.wg.Wait() // 等待所有 goroutine 退出
	log.Info("调度器已停止")
	return nil
}

// =============================================================================
// 快照采集循环
// =============================================================================

// runSnapshotLoop 快照采集循环
//
// 工作流程:
//  1. 启动时立即采集一次
//  2. 之后每隔 SnapshotInterval 采集一次
//  3. 采集失败只记录日志，不中断循环
func (s *Scheduler) runSnapshotLoop() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.config.SnapshotInterval)
	defer ticker.Stop()

	// 立即执行一次
	s.collectAndPushSnapshot()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.collectAndPushSnapshot()
		}
	}
}

// collectAndPushSnapshot 采集并推送快照
func (s *Scheduler) collectAndPushSnapshot() {
	ctx, cancel := context.WithTimeout(s.ctx, s.config.SnapshotTimeout)
	defer cancel()

	// 采集快照
	snapshot, err := s.snapshotSvc.Collect(ctx)
	if err != nil {
		log.Error("采集快照失败", "err", err)
		return
	}

	// 推送到 Master
	if err := s.masterGw.PushSnapshot(ctx, snapshot); err != nil {
		log.Error("推送快照失败", "err", err)
		return
	}

	// 检测资源数量变化
	podCount := len(snapshot.Pods)
	nodeCount := len(snapshot.Nodes)
	deploymentCount := len(snapshot.Deployments)

	if podCount != s.lastPodCount || nodeCount != s.lastNodeCount || deploymentCount != s.lastDeploymentCount {
		// 资源数量变化，输出 INFO 日志
		log.Info("快照已推送（资源变化）",
			"pods", podCount,
			"nodes", nodeCount,
			"deployments", deploymentCount,
		)
		s.lastPodCount = podCount
		s.lastNodeCount = nodeCount
		s.lastDeploymentCount = deploymentCount
	} else {
		// 无变化，输出 DEBUG 日志
		log.Debug("快照已推送",
			"pods", podCount,
			"nodes", nodeCount,
			"deployments", deploymentCount,
		)
	}
}

// =============================================================================
// 指令轮询循环
// =============================================================================

// runCommandLoop 指令轮询循环
//
// 每个 topic 独立运行一个循环，互不阻塞。
// 工作流程:
//  1. 长轮询获取 Master 指令 (超时 60s)
//  2. 执行每个指令
//  3. 上报执行结果
//  4. 短暂等待后继续轮询
func (s *Scheduler) runCommandLoop(topic string) {
	defer s.wg.Done()

	for {
		select {
		case <-s.ctx.Done():
			return
		default:
			s.pollAndExecuteCommands(topic)
			// 短暂等待后继续
			select {
			case <-s.ctx.Done():
				return
			case <-time.After(s.config.CommandPollInterval):
			}
		}
	}
}

// pollAndExecuteCommands 轮询并执行指令
func (s *Scheduler) pollAndExecuteCommands(topic string) {
	ctx, cancel := context.WithTimeout(s.ctx, s.config.CommandPollTimeout)
	defer cancel()

	// 拉取指令
	commands, err := s.masterGw.PollCommands(ctx, topic)
	if err != nil {
		log.Warn("拉取指令失败", "topic", topic, "err", err)
		return
	}

	if len(commands) == 0 {
		return
	}

	log.Info("收到指令", "count", len(commands), "topic", topic)

	// 并发执行所有指令
	var wg sync.WaitGroup
	for i := range commands {
		cmd := &commands[i]
		wg.Add(1)
		go func() {
			defer wg.Done()
			result := s.commandSvc.Execute(ctx, cmd)
			// 上报结果
			if err := s.masterGw.ReportResult(ctx, result); err != nil {
				log.Error("上报执行结果失败", "cmd_id", cmd.ID, "err", err)
			} else {
				log.Debug("指令已执行", "cmd_id", cmd.ID, "success", result.Success)
			}
		}()
	}
	wg.Wait()
}

// =============================================================================
// 心跳循环
// =============================================================================

// runHeartbeatLoop 心跳循环
//
// 定时向 Master 发送心跳，让 Master 知道 Agent 存活
func (s *Scheduler) runHeartbeatLoop() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.config.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(s.ctx, s.config.HeartbeatTimeout)
			if err := s.masterGw.Heartbeat(ctx); err != nil {
				log.Warn("心跳发送失败", "err", err)
			}
			cancel()
		}
	}
}

// =============================================================================
// SLO 采集循环
// =============================================================================

// runSLOLoop SLO 指标采集循环
//
// 工作流程:
//  1. 启动时等待一个采集周期（避免与快照采集冲突）
//  2. 之后每隔 SLOScrapeInterval 采集一次
//  3. 采集失败只记录日志，不中断循环
func (s *Scheduler) runSLOLoop() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.config.SLOScrapeInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.collectAndPushSLO()
		}
	}
}

// collectAndPushSLO 采集并推送 SLO 指标
func (s *Scheduler) collectAndPushSLO() {
	ctx, cancel := context.WithTimeout(s.ctx, s.config.SLOScrapeTimeout)
	defer cancel()

	// 采集指标
	metrics, err := s.sloSvc.Collect(ctx)
	if err != nil {
		log.Warn("SLO 指标采集失败", "err", err)
		return
	}

	if metrics == nil {
		return // 没有配置指标 URL
	}

	// 采集 IngressRoute 映射
	routes, err := s.sloSvc.CollectRoutes(ctx)
	if err != nil {
		log.Warn("IngressRoute 采集失败", "err", err)
		// 继续推送 metrics，routes 可以为空
	}

	// 推送到 Master
	if err := s.masterGw.PushSLOMetrics(ctx, metrics, routes); err != nil {
		log.Warn("SLO 指标推送失败", "err", err)
		return
	}

	log.Debug("SLO 指标已推送",
		"counters", len(metrics.Counters),
		"histograms", len(metrics.Histograms),
		"routes", len(routes),
	)
}

package scheduler

import (
	"context"
	"errors"
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

	// SnapshotPushTimeout 快照推送操作超时
	// 独立于 SnapshotTimeout，避免 Collect 慢耗尽 ctx 预算后连累 Push
	SnapshotPushTimeout time.Duration

	// CommandPollTimeout 指令轮询操作超时
	CommandPollTimeout time.Duration

	// CommandExecTimeout 单条指令的执行超时
	// 独立于 CommandPollTimeout：长轮询等了多久，不影响指令能执行多久
	CommandExecTimeout time.Duration

	// CommandReportTimeout 指令结果回报超时
	// 独立于执行超时：执行超时后仍要能把结果报回去
	CommandReportTimeout time.Duration

	// HeartbeatTimeout 心跳操作超时
	HeartbeatTimeout time.Duration
}

// Scheduler 调度器
//
// 管理 Agent 各项后台任务的生命周期:
//   - 快照采集循环 - 定时采集集群资源（含 SLO 数据），推送给 Master
//   - 指令轮询循环 - 长轮询获取 Master 指令，执行后上报结果
//   - 心跳循环 - 定时向 Master 发送心跳，维持连接状态
type Scheduler struct {
	config Config

	// 依赖的服务
	snapshotSvc service.SnapshotService
	commandSvc  service.CommandService
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
//   - snapshotSvc: 快照服务，用于采集集群资源（含 SLO 数据）
//   - commandSvc: 指令服务，用于执行 Master 下发的指令
//   - masterGw: Master 网关，用于与 Master 通信
func New(
	config Config,
	snapshotSvc service.SnapshotService,
	commandSvc service.CommandService,
	masterGw gateway.MasterGateway,
) *Scheduler {
	return &Scheduler{
		config:      config,
		snapshotSvc: snapshotSvc,
		commandSvc:  commandSvc,
		masterGw:    masterGw,
	}
}

// Start 启动调度器
func (s *Scheduler) Start(ctx context.Context) error {
	s.ctx, s.cancel = context.WithCancel(ctx)

	// 启动后台任务: 快照 + ops轮询 + ai轮询 + 心跳
	s.wg.Add(4)
	go s.runSnapshotLoop()     // 快照采集（含 OTel 概览数据）
	go s.runCommandLoop("ops") // 系统操作指令轮询
	go s.runCommandLoop("ai")  // AI 查询指令轮询
	go s.runHeartbeatLoop()    // 心跳

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
//
// Collect 和 Push 使用独立 ctx，互不干扰：
//   - Collect ctx 超时只影响采集本身
//   - Push ctx 拥有全新的超时预算，即使 Collect 耗时较长也能正常发送
func (s *Scheduler) collectAndPushSnapshot() {
	// 采集快照（独立超时）
	collectCtx, cancelCollect := context.WithTimeout(s.ctx, s.config.SnapshotTimeout)
	snapshot, err := s.snapshotSvc.Collect(collectCtx)
	cancelCollect()
	if err != nil {
		log.Error("采集快照失败", "err", err)
		return
	}

	// 推送到 Master（独立超时，不受 Collect 耗时影响）
	pushCtx, cancelPush := context.WithTimeout(s.ctx, s.config.SnapshotPushTimeout)
	defer cancelPush()
	if err := s.masterGw.PushSnapshot(pushCtx, snapshot); err != nil {
		log.Error(masterFailureMessage("推送快照失败", err), "err", err)
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
//  4. 有指令时立即再 poll（排空队列），无指令时等待间隔
func (s *Scheduler) runCommandLoop(topic string) {
	defer s.wg.Done()

	backoff := s.config.CommandPollInterval
	maxBackoff := 30 * time.Second
	var failCount int
	var lastWarnTime time.Time

	for {
		select {
		case <-s.ctx.Done():
			return
		default:
			ok, pollErr := s.pollAndExecuteCommands(topic)
			if ok {
				backoff = s.config.CommandPollInterval
				failCount = 0
				continue // 有指令，立即再 poll（排空队列）
			}

			if pollErr != nil {
				failCount++
				// 首次失败 + 之后每 30s 打一次 WARN，避免刷屏。
				// 身份被拒也只记录并继续重试、不退出：令牌在 Master 侧补签或登记后即可自行恢复
				if failCount == 1 || time.Since(lastWarnTime) >= 30*time.Second {
					log.Warn(masterFailureMessage("拉取指令失败，等待重试", pollErr),
						"topic", topic, "consecutiveFails", failCount, "err", pollErr)
					lastWarnTime = time.Now()
				}
				// 指数退避: 100ms → 200ms → 400ms → ... → 30s
				backoff = min(backoff*2, maxBackoff)
			} else {
				backoff = s.config.CommandPollInterval
				if failCount > 0 {
					log.Info("指令轮询恢复", "topic", topic, "previousFails", failCount)
					failCount = 0
				}
			}

			select {
			case <-s.ctx.Done():
				return
			case <-time.After(backoff):
			}
		}
	}
}

// pollAndExecuteCommands 轮询并执行指令
// 返回 (hasCommands, pollErr)；pollErr 非 nil 表示这一轮没能从 Master 拉到指令，由调用方限频记录并退避
func (s *Scheduler) pollAndExecuteCommands(topic string) (bool, error) {
	// 三个阶段各用各的时间预算：长轮询可能等到快超时才拿到指令，
	// 执行与回报不能只剩它用剩下的那点时间。
	pollCtx, cancelPoll := context.WithTimeout(s.ctx, s.config.CommandPollTimeout)
	commands, err := s.masterGw.PollCommands(pollCtx, topic)
	cancelPoll()
	if err != nil {
		return false, err
	}

	if len(commands) == 0 {
		return false, nil
	}

	// 并发执行所有指令
	var wg sync.WaitGroup
	for i := range commands {
		cmd := &commands[i]
		wg.Add(1)
		go func() {
			defer wg.Done()
			start := time.Now()
			execCtx, cancelExec := context.WithTimeout(s.ctx, durationOr(s.config.CommandExecTimeout, defaultCommandExecTimeout))
			result := s.commandSvc.Execute(execCtx, cmd)
			cancelExec()
			elapsed := time.Since(start)

			// 构建可读的 action 标识
			action := cmd.Action
			if sub, ok := cmd.Params["sub_action"].(string); ok && sub != "" {
				action += "/" + sub
			}

			// 上报结果（独立预算：执行超时后也要能报回去）
			reportCtx, cancelReport := context.WithTimeout(s.ctx, durationOr(s.config.CommandReportTimeout, defaultCommandReportTimeout))
			defer cancelReport()
			if err := s.masterGw.ReportResult(reportCtx, result); err != nil {
				log.Error(masterFailureMessage("指令结果回报失败", err), "action", action, "elapsed", elapsed.Round(time.Millisecond), "err", err)
			} else {
				log.Info("指令完成", "action", action, "elapsed", elapsed.Round(time.Millisecond), "success", result.Success)
			}
		}()
	}
	wg.Wait()
	return true, nil
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
				log.Warn(masterFailureMessage("心跳发送失败", err), "err", err)
			}
			cancel()
		}
	}
}

// 指令执行 / 回报超时的兜底值（配置未给出时使用）
const (
	defaultCommandExecTimeout   = 60 * time.Second
	defaultCommandReportTimeout = 10 * time.Second
)

func durationOr(configured, fallback time.Duration) time.Duration {
	if configured > 0 {
		return configured
	}
	return fallback
}

// masterFailureMessage 给「与 Master 通信失败」配日志文案。
//
// 身份被拒（401 / 403）与网络故障的处理办法完全不同：前者要去检查令牌与集群登记，
// 后者等网络恢复即可。文案里直接点明，排查时不必去读错误详情。
func masterFailureMessage(action string, err error) string {
	if errors.Is(err, gateway.ErrMasterRejectedIdentity) {
		return action + "：Master 拒绝了本 Agent 的身份（认证问题，不是网络故障）"
	}
	return action
}

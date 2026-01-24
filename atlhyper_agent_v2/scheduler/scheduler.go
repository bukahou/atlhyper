package scheduler

import (
	"context"
	"log"
	"sync"
	"time"

	"AtlHyper/atlhyper_agent_v2/gateway"
	"AtlHyper/atlhyper_agent_v2/service"
)

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
}

// Scheduler 调度器
//
// 管理 Agent 各项后台任务的生命周期:
//   - 快照采集循环 - 定时采集集群资源，推送给 Master
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
}

// New 创建调度器
//
// 参数:
//   - config: 调度器配置
//   - snapshotSvc: 快照服务，用于采集集群资源
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

	// 启动后台任务
	s.wg.Add(4)
	go s.runSnapshotLoop()       // 快照采集
	go s.runCommandLoop("ops")   // 系统操作指令轮询
	go s.runCommandLoop("ai")    // AI 查询指令轮询
	go s.runHeartbeatLoop()      // 心跳

	log.Println("[Scheduler] 调度器已启动")
	return nil
}

// Stop 停止调度器
func (s *Scheduler) Stop() error {
	if s.cancel != nil {
		s.cancel() // 通知所有 goroutine 停止
	}
	s.wg.Wait() // 等待所有 goroutine 退出
	log.Println("[Scheduler] 调度器已停止")
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
		log.Printf("[Scheduler] 采集快照失败: %v", err)
		return
	}

	// 推送到 Master
	if err := s.masterGw.PushSnapshot(ctx, snapshot); err != nil {
		log.Printf("[Scheduler] 推送快照失败: %v", err)
		return
	}

	log.Printf("[Scheduler] 快照已推送: Pods=%d, Nodes=%d, Deployments=%d",
		len(snapshot.Pods), len(snapshot.Nodes), len(snapshot.Deployments))
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
		log.Printf("[Scheduler] 拉取指令失败 [%s]: %v", topic, err)
		return
	}

	if len(commands) == 0 {
		return
	}

	log.Printf("[Scheduler] 收到 %d 条指令 [%s]", len(commands), topic)

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
				log.Printf("[Scheduler] 上报执行结果失败: 指令=%s, 错误=%v", cmd.ID, err)
			} else {
				log.Printf("[Scheduler] 指令已执行: id=%s, 成功=%v", cmd.ID, result.Success)
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
				log.Printf("[Scheduler] 心跳发送失败: %v", err)
			}
			cancel()
		}
	}
}

// atlhyper_master_v2/notifier/pipeline/pipeline.go
// 通知管道：告警入队 → 路由 → 静默 → 抑制 → 分组 → 重复提醒 → 发送与重试 → 送达记录
//
// 并发模型：全部状态由一个后台协程独占，不加锁。外部只通过 Submit / SubmitDeferred 往有界队列里放东西，
// 放不进去就丢弃并计数 —— 调用方（快照处理主路径）永远不会被通知拖住。
package pipeline

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"sync"
	"sync/atomic"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/notifier/alert"
	"AtlHyper/atlhyper_master_v2/notifier/channel"
	"AtlHyper/atlhyper_master_v2/notifier/template"
	"AtlHyper/common/logger"
)

var log = logger.Module("NotifyPipeline")

const (
	defaultQueueSize       = 1024
	defaultTickInterval    = time.Second
	defaultSendConcurrency = 4

	// storeTimeout 单次读写库的超时
	storeTimeout = 10 * time.Second
	// sendTimeout 单次发送的超时上限（各发送器自己还有更短的超时）
	sendTimeout = 45 * time.Second
	// retentionPeriod 送达记录、已结束的告警、已过期的静默的保留时间
	retentionPeriod = 30 * 24 * time.Hour
	// cleanupInterval 清理任务的间隔
	cleanupInterval = time.Hour
	// dropLogInterval 「队列已满」这条日志的最短间隔
	dropLogInterval = time.Minute
	// maxLastErrorLength 写进送达记录的错误信息的长度上限
	maxLastErrorLength = 500
	// noReceiverWarnWindow 「没有任何可用的接收人」这条日志，同一个集群的最短间隔
	noReceiverWarnWindow = 10 * time.Minute

	// SuppressedByClusterOffline 抑制原因：该集群的 Agent 离线告警正在持续
	SuppressedByClusterOffline = "cluster-offline"
)

// retryBackoff 发送失败后的重试间隔：30 秒、2 分钟、10 分钟；三次重试都失败即放弃
var retryBackoff = []time.Duration{30 * time.Second, 2 * time.Minute, 10 * time.Minute}

// Sender 把一条消息发给一个接收人。返回的错误不得带接收人的凭据
type Sender interface {
	Send(ctx context.Context, receiver *database.NotifyReceiver, msg *channel.Message) error
}

// Renderer 把一批告警渲染成发给某类接收人的一条消息
type Renderer interface {
	Render(receiverType string, items []template.Item, now time.Time) (*channel.Message, error)
}

// Repos 管道读写的仓库
type Repos struct {
	Policy     database.NotifyPolicyRepository
	Alerts     database.NotifyAlertRepository
	Deliveries database.NotifyDeliveryRepository
	Receivers  database.NotifyReceiverRepository
	Rules      database.NotifyRuleRepository
	Silences   database.NotifySilenceRepository
}

// Config 管道的依赖与参数。时钟与发送器可替换，便于测试
type Config struct {
	Repos    Repos
	Sender   Sender
	Renderer Renderer
	// Now 时钟；nil = time.Now
	Now func() time.Time
	// QueueSize 入队队列的容量；0 = 1024
	QueueSize int
	// TickInterval 后台协程多久跑一轮（分组到期、重试、重复提醒都在这一轮里检查）；0 = 1 秒
	TickInterval time.Duration
	// SendConcurrency 同一轮里并行发送的批次数上限；0 = 4
	SendConcurrency int
}

// submission 队列里的一项：一条告警，或一个「拿到当前策略后再产出告警」的构造函数
type submission struct {
	alert *alert.Alert
	build func(policy *database.NotifyPolicy) (alert.Alert, bool)
}

// AlertPipeline 通知管道
type AlertPipeline struct {
	repos           Repos
	sender          Sender
	renderer        Renderer
	now             func() time.Time
	tickInterval    time.Duration
	sendConcurrency int

	queue       chan submission
	dropped     atomic.Uint64
	lastDropLog atomic.Int64 // 上一次记「队列已满」日志的时刻（UnixNano）

	lifecycle sync.Mutex
	started   bool
	stopped   atomic.Bool
	stopCh    chan struct{}
	done      chan struct{}

	// 以下状态由后台协程独占（测试里由测试协程独占），不加锁
	alerts        map[string]*alertState   // 持续中的告警，以及恢复通知还没发出去的已结束告警
	groups        map[groupKey]*groupState // 分组：同一条规则 + 同一个集群
	batches       map[string]*batchState   // 还没有定论的批次（待发 / 等重试）
	batchSeq      int                      // 批次的创建序号，保证处理顺序稳定
	lastCleanupAt time.Time

	// noReceiverWarnings 「没有任何可用的接收人」日志的限频（按集群）
	noReceiverWarnings *repeatLogLimiter
}

// NewAlertPipeline 创建通知管道
func NewAlertPipeline(cfg Config) *AlertPipeline {
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	if cfg.QueueSize <= 0 {
		cfg.QueueSize = defaultQueueSize
	}
	if cfg.TickInterval <= 0 {
		cfg.TickInterval = defaultTickInterval
	}
	if cfg.SendConcurrency <= 0 {
		cfg.SendConcurrency = defaultSendConcurrency
	}
	return &AlertPipeline{
		repos:           cfg.Repos,
		sender:          cfg.Sender,
		renderer:        cfg.Renderer,
		now:             cfg.Now,
		tickInterval:    cfg.TickInterval,
		sendConcurrency: cfg.SendConcurrency,
		queue:           make(chan submission, cfg.QueueSize),
		stopCh:          make(chan struct{}),
		done:            make(chan struct{}),
		alerts:          make(map[string]*alertState),
		groups:          make(map[groupKey]*groupState),
		batches:         make(map[string]*batchState),

		noReceiverWarnings: newRepeatLogLimiter(noReceiverWarnWindow, cfg.Now),
	}
}

// Start 从库里恢复状态并启动后台协程。重复调用无效果
func (p *AlertPipeline) Start() error {
	p.lifecycle.Lock()
	defer p.lifecycle.Unlock()
	if p.started {
		return nil
	}
	p.started = true

	p.restoreFromStore()
	go p.loop()
	log.Info("通知管道已启动", "queueSize", cap(p.queue), "tickInterval", p.tickInterval.String())
	return nil
}

// Stop 停止后台协程并等它退出。停止后不再接受新的告警；重复调用无效果
func (p *AlertPipeline) Stop() {
	p.lifecycle.Lock()
	defer p.lifecycle.Unlock()
	if !p.started || p.stopped.Load() {
		p.stopped.Store(true)
		return
	}
	p.stopped.Store(true)
	close(p.stopCh)
	<-p.done
	log.Info("通知管道已停止", "droppedTotal", p.dropped.Load())
}

func (p *AlertPipeline) loop() {
	defer close(p.done)
	ticker := time.NewTicker(p.tickInterval)
	defer ticker.Stop()
	for {
		select {
		case <-p.stopCh:
			return
		case item := <-p.queue:
			p.process(item)
		case <-ticker.C:
			p.tick()
		}
	}
}

// Submit 提交一条告警。只做入队，立即返回；队列满（或管道已停止）时丢弃并返回 false
func (p *AlertPipeline) Submit(a alert.Alert) bool {
	return p.offer(submission{alert: &a})
}

// SubmitDeferred 提交一个构造函数：后台协程读到当前策略后调用它，由它决定是否产出告警。
//
// 给「是否产生告警取决于策略」的来源用（事件告警的开关、警告级反复的阈值）：
// 调用方在快照处理主路径上，不能自己去读库
func (p *AlertPipeline) SubmitDeferred(build func(policy *database.NotifyPolicy) (alert.Alert, bool)) bool {
	return p.offer(submission{build: build})
}

// Dropped 因队列已满（或管道已停止）而丢弃的提交总数
func (p *AlertPipeline) Dropped() uint64 { return p.dropped.Load() }

func (p *AlertPipeline) offer(item submission) bool {
	if p.stopped.Load() {
		p.recordDrop()
		return false
	}
	select {
	case p.queue <- item:
		return true
	default:
		p.recordDrop()
		return false
	}
}

// recordDrop 丢弃计数，并限频记日志：队列满往往是一阵一阵的，逐条记只会刷屏
func (p *AlertPipeline) recordDrop() {
	total := p.dropped.Add(1)
	current := time.Now().UnixNano()
	last := p.lastDropLog.Load()
	if current-last < int64(dropLogInterval) {
		return
	}
	if p.lastDropLog.CompareAndSwap(last, current) {
		log.Warn("通知队列已满或管道已停止，告警被丢弃", "droppedTotal", total, "logInterval", dropLogInterval.String())
	}
}

// process 处理队列里的一项：落成告警状态，然后对有变化的告警做路由
func (p *AlertPipeline) process(item submission) {
	now := p.now()
	current, err := p.loadSettings()

	incoming := item.alert
	if item.build != nil {
		if err != nil {
			log.Error("读取通知配置失败，本次事件变化未处理", "err", err)
			return
		}
		built, produced := item.build(current.policy)
		if !produced {
			return
		}
		incoming = &built
	}

	p.apply(*incoming, now)
	if err != nil {
		// 告警已经落下，待通知的标记还在：下一轮读到配置后再路由
		log.Error("读取通知配置失败，稍后重试路由", "err", err)
		return
	}
	p.routePending(current, now)
}

// tick 一轮定时检查
func (p *AlertPipeline) tick() {
	now := p.now()
	defer p.cleanupIfDue(now)
	if p.idle() {
		return
	}
	current, err := p.loadSettings()
	if err != nil {
		log.Error("读取通知配置失败，本轮跳过", "err", err)
		return
	}
	p.refreshHolds(current, now)
	p.scheduleRepeats(current, now)
	p.routePending(current, now)
	p.flushDueGroups(current, now)
	p.sendDueBatches(current, now)
	p.pruneIdleGroups(now)
}

// idle 没有任何告警、分组里没有待发内容、也没有待处理的批次
func (p *AlertPipeline) idle() bool {
	return len(p.alerts) == 0 && len(p.batches) == 0
}

// storeContext 读写库用的上下文
func (p *AlertPipeline) storeContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), storeTimeout)
}

// newBatchID 批次号：b- 加 16 位随机十六进制
func newBatchID() string {
	raw := make([]byte, 8)
	if _, err := rand.Read(raw); err != nil {
		return "b-" + hex.EncodeToString([]byte(time.Now().Format("150405.000000")))
	}
	return "b-" + hex.EncodeToString(raw)
}

// atlhyper_master_v2/notifier/manager.go
// AlertManager 实现：装配告警来源、通知管道与发送器
package notifier

import (
	"context"
	"net/url"
	"strings"
	"time"
	_ "time/tzdata" // 镜像里不一定有系统时区库：内嵌一份，MASTER_TIMEZONE 才可用

	"AtlHyper/atlhyper_master_v2/aiops"
	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/notifier/alert"
	"AtlHyper/atlhyper_master_v2/notifier/channel"
	"AtlHyper/atlhyper_master_v2/notifier/pipeline"
	"AtlHyper/atlhyper_master_v2/notifier/source"
	"AtlHyper/atlhyper_master_v2/notifier/template"
	"AtlHyper/common/logger"
)

var log = logger.Module("Notifier")

// Config 通知模块的依赖与参数
type Config struct {
	Repos  pipeline.Repos             // 通知各表的仓库
	SMTP   channel.SMTPSettingsReader // 全局发件设置（邮件接收人发送时读取）
	Events source.ClusterEventReader  // 持久化的 K8s Warning 事件（旧的逐条事件告警用）
	Agents source.AgentLister         // 各集群 Agent 的在线状态（心跳告警用）

	// PublicURL Master 的对外地址（MASTER_PUBLIC_URL）；空或不合法时通知里不带页面链接
	PublicURL string
	// Location 通知里时间的显示时区（由 MASTER_TIMEZONE 解析而来）；nil = UTC
	Location *time.Location

	HeartbeatCheckInterval time.Duration // 心跳检测间隔；0 = 30 秒
	HeartbeatOfflineAfter  time.Duration // 心跳多久没更新算离线；0 = 60 秒
	K8sEventCheckInterval  time.Duration // K8s 事件轮询间隔；0 = 30 秒

	// 以下两项只在测试里设置
	Sender               pipeline.Sender // nil = 按接收人的配置真实发送
	PipelineTickInterval time.Duration   // 0 = 1 秒
}

// Manager AlertManager 的实现
type Manager struct {
	pipeline  *pipeline.AlertPipeline
	incidents *source.IncidentSource
	heartbeat *source.HeartbeatSource
	k8sEvents *source.K8sEventSource

	receivers receiverReader
	renderer  *template.AlertRenderer
	sender    pipeline.Sender
}

// receiverReader 测试发送只需要按 id 取接收人
type receiverReader interface {
	GetByID(ctx context.Context, id int64) (*database.NotifyReceiver, error)
}

// NewAlertManager 创建通知模块（不启动任何协程，启动见 Start）
func NewAlertManager(cfg Config) (*Manager, error) {
	publicURL := sanitizePublicURL(cfg.PublicURL)
	if cfg.PublicURL != "" && publicURL == "" {
		log.Warn("MASTER_PUBLIC_URL 不是合法的 http(s) 地址，通知里不会带页面链接")
	}
	location := cfg.Location
	if location == nil {
		location = time.UTC
	}
	renderer, err := template.NewAlertRenderer(publicURL, location)
	if err != nil {
		return nil, err
	}
	sender := cfg.Sender
	if sender == nil {
		sender = channel.NewReceiverSender(cfg.SMTP)
	}

	alertPipeline := pipeline.NewAlertPipeline(pipeline.Config{
		Repos:        cfg.Repos,
		Sender:       sender,
		Renderer:     renderer,
		TickInterval: cfg.PipelineTickInterval,
	})
	return &Manager{
		pipeline:  alertPipeline,
		incidents: source.NewIncidentSource(alertPipeline),
		heartbeat: source.NewHeartbeatSource(cfg.Agents, cfg.Repos.Alerts, alertPipeline, source.HeartbeatConfig{
			CheckInterval: cfg.HeartbeatCheckInterval,
			OfflineAfter:  cfg.HeartbeatOfflineAfter,
		}),
		k8sEvents: source.NewK8sEventSource(cfg.Events, cfg.Repos.Policy, cfg.Repos.Alerts, alertPipeline, source.K8sEventConfig{
			CheckInterval: cfg.K8sEventCheckInterval,
		}),
		receivers: cfg.Repos.Receivers,
		renderer:  renderer,
		sender:    sender,
	}, nil
}

// Start 先启动管道（恢复状态），再启动两个轮询来源。重复调用无效果
func (m *Manager) Start() error {
	if err := m.pipeline.Start(); err != nil {
		return err
	}
	if err := m.heartbeat.Start(); err != nil {
		return err
	}
	if err := m.k8sEvents.Start(); err != nil {
		return err
	}
	log.Info("通知模块已启动")
	return nil
}

// Stop 先停来源，再停管道。重复调用无效果
func (m *Manager) Stop() {
	if err := m.k8sEvents.Stop(); err != nil {
		log.Error("停止 K8s 事件告警来源失败", "err", err)
	}
	if err := m.heartbeat.Stop(); err != nil {
		log.Error("停止心跳告警来源失败", "err", err)
	}
	m.pipeline.Stop()
	log.Info("通知模块已停止")
}

// OnIncidentTransition 事件变化的订阅回调：只入队
func (m *Manager) OnIncidentTransition(transition aiops.IncidentTransition) {
	m.incidents.OnTransition(transition)
}

// OnIncidentLink 拓扑归并的关联变化：只入队
func (m *Manager) OnIncidentLink(link aiops.IncidentLink) {
	root := ""
	if link.RootIncidentID != "" {
		root = alert.IncidentFingerprint(link.RootIncidentID)
	}
	m.pipeline.SubmitLink(alert.IncidentFingerprint(link.IncidentID), root, link.Subject)
}

// TestReceiver 给指定接收人发一条测试消息。不经过管道：不产生告警，也不留送达记录
func (m *Manager) TestReceiver(ctx context.Context, receiverID int64) error {
	receiver, err := m.receivers.GetByID(ctx, receiverID)
	if err != nil {
		return err
	}
	if receiver == nil {
		return ErrReceiverNotFound
	}
	return m.sender.Send(ctx, receiver, m.renderer.RenderTest(receiver.Name, time.Now()))
}

// sanitizePublicURL 只接受带主机的 http / https 地址，去掉末尾的斜杠；其余一律当作没配
func sanitizePublicURL(raw string) string {
	trimmed := strings.TrimRight(strings.TrimSpace(raw), "/")
	if trimmed == "" {
		return ""
	}
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return ""
	}
	return trimmed
}

var _ AlertManager = (*Manager)(nil)

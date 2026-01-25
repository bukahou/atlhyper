// atlhyper_master_v2/notifier/manager.go
// 告警管理器实现
// 编排 template 和 channel 模块
package notifier

import (
	"context"
	"log"
	"sync"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/notifier/channel"
	"AtlHyper/atlhyper_master_v2/notifier/template"
)

// Manager 告警管理器
type Manager struct {
	notifyRepo database.NotifyChannelRepository
	factory    *channel.Factory
	renderer   *template.Renderer

	mu sync.RWMutex
}

// NewManager 创建告警管理器
func NewManager(notifyRepo database.NotifyChannelRepository) (*Manager, error) {
	renderer, err := template.NewRenderer()
	if err != nil {
		return nil, err
	}

	return &Manager{
		notifyRepo: notifyRepo,
		factory:    channel.NewFactory(),
		renderer:   renderer,
	}, nil
}

// Start 启动
func (m *Manager) Start() error {
	log.Println("[AlertManager] 已启动")
	return nil
}

// Stop 停止
func (m *Manager) Stop() {
	log.Println("[AlertManager] 已停止")
}

// SendWithTemplate 使用模板发送告警
func (m *Manager) SendWithTemplate(templateName string, data *template.AlertData) error {
	ctx := context.Background()

	// 1. 获取所有已启用的渠道
	channels, err := m.notifyRepo.ListEnabled(ctx)
	if err != nil {
		log.Printf("[AlertManager] 获取渠道列表失败: %v", err)
		return err
	}

	if len(channels) == 0 {
		log.Printf("[AlertManager] 无可用渠道，告警未发送: %s", data.Title)
		return nil
	}

	// 2. 并发发送到各渠道
	var wg sync.WaitGroup
	for _, ch := range channels {
		wg.Add(1)
		go func(ch *database.NotifyChannel) {
			defer wg.Done()

			// 渲染模板
			msg, err := m.renderer.Render(templateName, ch.Type, data)
			if err != nil {
				log.Printf("[AlertManager] 渲染模板失败: channel=%s, err=%v", ch.Type, err)
				return
			}

			// 创建 notifier
			notifier, err := m.factory.Create(ch)
			if err != nil {
				log.Printf("[AlertManager] 创建通知器失败: channel=%s, err=%v", ch.Type, err)
				return
			}

			// 发送
			if err := notifier.Send(ctx, msg); err != nil {
				log.Printf("[AlertManager] 发送失败: channel=%s, err=%v", ch.Type, err)
				return
			}

			log.Printf("[AlertManager] 发送成功: channel=%s, title=%s", ch.Type, data.Title)
		}(ch)
	}
	wg.Wait()

	return nil
}

// Test 测试指定渠道
func (m *Manager) Test(ctx context.Context, channelType string) error {
	// 获取渠道配置
	ch, err := m.notifyRepo.GetByType(ctx, channelType)
	if err != nil {
		return ErrChannelNotFound
	}

	if !ch.Enabled {
		return ErrChannelDisabled
	}

	// 创建 notifier
	notifier, err := m.factory.Create(ch)
	if err != nil {
		return ErrInvalidConfig
	}

	// 发送测试消息
	testData := &template.AlertData{
		Title:     "测试告警",
		Message:   "这是一条测试告警消息，用于验证通知渠道配置是否正确。",
		Severity:  "info",
		Source:    "manual",
		ClusterID: "test",
		Resource:  "test/test/test",
		Reason:    "Test",
	}

	msg, err := m.renderer.Render("heartbeat_recovery", channelType, testData)
	if err != nil {
		return err
	}

	return notifier.Send(ctx, msg)
}

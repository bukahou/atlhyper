// atlhyper_master_v2/database/repository/notify_channel.go
// NotifyChannelRepository 接口定义
package repository

import (
	"context"
	"time"
)

// NotifyChannel 通知渠道
type NotifyChannel struct {
	ID        int64
	Type      string // slack / email（UNIQUE，一个类型一条记录）
	Name      string // 显示名称
	Enabled   bool   // 是否启用（默认 false）
	Config    string // JSON 配置
	CreatedAt time.Time
	UpdatedAt time.Time
}

// SlackConfig Slack 配置
type SlackConfig struct {
	WebhookURL string `json:"webhook_url"`
}

// EmailConfig Email 配置
type EmailConfig struct {
	SMTPHost    string   `json:"smtp_host"`
	SMTPPort    int      `json:"smtp_port"`
	SMTPUser    string   `json:"smtp_user"`
	SMTPPassword string  `json:"smtp_password"`
	SMTPTLS     bool     `json:"smtp_tls"`
	FromAddress string   `json:"from_address"`
	ToAddresses []string `json:"to_addresses"`
}

// NotifyChannelRepository 通知渠道接口
type NotifyChannelRepository interface {
	// Create 创建渠道
	Create(ctx context.Context, channel *NotifyChannel) error

	// Update 更新渠道
	Update(ctx context.Context, channel *NotifyChannel) error

	// Delete 删除渠道
	Delete(ctx context.Context, id int64) error

	// GetByID 按 ID 获取
	GetByID(ctx context.Context, id int64) (*NotifyChannel, error)

	// GetByType 按类型获取（一个类型一条记录）
	GetByType(ctx context.Context, channelType string) (*NotifyChannel, error)

	// List 列出所有渠道
	List(ctx context.Context) ([]*NotifyChannel, error)

	// ListEnabled 列出已启用的渠道
	ListEnabled(ctx context.Context) ([]*NotifyChannel, error)
}

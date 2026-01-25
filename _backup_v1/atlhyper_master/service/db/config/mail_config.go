// service/config/mail_config.go
// 邮件配置服务
package config

import (
	"context"
	"time"

	"AtlHyper/atlhyper_master/repository"
)

// MailConfigDTO 邮件配置 DTO（供 Handler 使用）
type MailConfigDTO struct {
	ID          int64     `json:"id"`
	Enable      int       `json:"enable"`
	SMTPHost    string    `json:"smtpHost"`
	SMTPPort    string    `json:"smtpPort"`
	Username    string    `json:"username"`
	Password    string    `json:"password"`
	FromAddr    string    `json:"fromAddr"`
	ToAddrs     string    `json:"toAddrs"`
	IntervalSec int64     `json:"intervalSec"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// GetMailConfigUI 返回邮件配置（给 Web UI 使用）
func GetMailConfigUI(ctx context.Context) (MailConfigDTO, error) {
	cfg, err := repository.Config.GetMailConfig(ctx)
	if err != nil {
		return MailConfigDTO{}, err
	}
	enable := 0
	if cfg.Enable {
		enable = 1
	}
	return MailConfigDTO{
		ID:          cfg.ID,
		Enable:      enable,
		SMTPHost:    cfg.SMTPHost,
		SMTPPort:    cfg.SMTPPort,
		Username:    cfg.Username,
		Password:    cfg.Password,
		FromAddr:    cfg.FromAddr,
		ToAddrs:     cfg.ToAddrs,
		IntervalSec: cfg.IntervalSec,
		UpdatedAt:   cfg.UpdatedAt,
	}, nil
}

// GetMailConfigMasked 返回脱敏的邮件配置（低权限用户）
func GetMailConfigMasked() MailConfigDTO {
	return MailConfigDTO{
		ID:          1,
		Enable:      0,
		SMTPHost:    "smtp.***.com",
		SMTPPort:    "587",
		Username:    "***@***.com",
		Password:    "******",
		FromAddr:    "***@***.com",
		ToAddrs:     "***@***.com",
		IntervalSec: 60,
		UpdatedAt:   time.Time{},
	}
}

// MailUpdateReq Web 更新入参
type MailUpdateReq struct {
	Enable      *int    `json:"enable,omitempty"`
	SMTPHost    *string `json:"smtpHost,omitempty"`
	SMTPPort    *string `json:"smtpPort,omitempty"`
	Username    *string `json:"username,omitempty"`
	Password    *string `json:"password,omitempty"`
	FromAddr    *string `json:"fromAddr,omitempty"`
	ToAddrs     *string `json:"toAddrs,omitempty"`
	IntervalSec *int64  `json:"intervalSec,omitempty"`
}

// UpdateMailConfigUI 更新邮件配置
func UpdateMailConfigUI(ctx context.Context, req MailUpdateReq) error {
	// 先获取当前配置
	cfg, err := repository.Config.GetMailConfig(ctx)
	if err != nil {
		return err
	}

	// 根据请求更新字段
	if req.Enable != nil {
		cfg.Enable = *req.Enable != 0
	}
	if req.SMTPHost != nil {
		cfg.SMTPHost = *req.SMTPHost
	}
	if req.SMTPPort != nil {
		cfg.SMTPPort = *req.SMTPPort
	}
	if req.Username != nil {
		cfg.Username = *req.Username
	}
	if req.Password != nil {
		cfg.Password = *req.Password
	}
	if req.FromAddr != nil {
		cfg.FromAddr = *req.FromAddr
	}
	if req.ToAddrs != nil {
		cfg.ToAddrs = *req.ToAddrs
	}
	if req.IntervalSec != nil {
		cfg.IntervalSec = *req.IntervalSec
	}

	return repository.Config.UpdateMailConfig(ctx, cfg)
}

// atlhyper_master/config/types.go
// 配置结构体定义
package config

import "time"

// DiagnosisConfig 诊断系统配置
type DiagnosisConfig struct {
	CleanInterval            time.Duration // 清理器执行间隔
	WriteInterval            time.Duration // 日志写入间隔
	RetentionRawDuration     time.Duration // 原始事件保留时间
	RetentionCleanedDuration time.Duration // 清理池保留时间
	UnreadyThresholdDuration time.Duration // 告警与邮件发送时间间隔
	AlertDispatchInterval    time.Duration // 邮件轮询检测发送间隔
	UnreadyReplicaPercent    float64
}

// KubernetesConfig Kubernetes API 健康检查配置
type KubernetesConfig struct {
	APIHealthCheckInterval time.Duration
}

// MailerConfig 邮件发送配置
type MailerConfig struct {
	SMTPHost         string
	SMTPPort         string
	Username         string
	Password         string
	From             string
	To               []string
	EnableEmailAlert bool
}

// SlackConfig Slack 通知配置
type SlackConfig struct {
	WebhookURL       string
	DispatchInterval time.Duration
	EnableSlackAlert bool
}

// WebhookConfig Webhook 配置
type WebhookConfig struct {
	Enable bool
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port string
}

// AdminConfig 管理员初始配置
type AdminConfig struct {
	Username    string
	Password    string
	DisplayName string
	Email       string
	Role        string
}

// StoreConfig 内存存储配置（master_store）
type StoreConfig struct {
	TTL             time.Duration // 每条记录的默认生存时间
	MaxItems        int           // 全局池最多保留的记录数
	CleanupInterval time.Duration // 清理任务的执行间隔
	MetricsTTL      time.Duration // 指标数据的 TTL（通常比事件短）
}

// JWTConfig JWT 认证配置
type JWTConfig struct {
	SecretKey       string        // JWT 签名密钥
	TokenExpiry     time.Duration // Token 有效期
	MinPasswordLen  int           // 密码最小长度
}

// AppConfig Master 顶层配置结构体
type AppConfig struct {
	Diagnosis  DiagnosisConfig
	Kubernetes KubernetesConfig
	Mailer     MailerConfig
	Slack      SlackConfig
	Webhook    WebhookConfig
	Server     ServerConfig
	Admin      AdminConfig
	Store      StoreConfig
	JWT        JWTConfig
}

// GlobalConfig 全局配置实例
var GlobalConfig AppConfig

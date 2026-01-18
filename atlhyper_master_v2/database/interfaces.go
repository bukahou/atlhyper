// atlhyper_master_v2/database/interfaces.go
// Database 接口定义
// Database 是持久化数据层，负责 RDB 操作
package database

import "AtlHyper/atlhyper_master_v2/database/repository"

// Database 数据库接口
// 抽象层，底层可替换为 SQLite / MySQL / PostgreSQL
type Database interface {
	// ==================== Repository 获取 ====================

	// ClusterEventRepository Event 持久化
	ClusterEventRepository() repository.ClusterEventRepository

	// NotifyChannelRepository 通知渠道
	NotifyChannelRepository() repository.NotifyChannelRepository

	// UserRepository 用户管理
	UserRepository() repository.UserRepository

	// ClusterRepository 集群管理
	ClusterRepository() repository.ClusterRepository

	// AuditRepository 审计日志
	AuditRepository() repository.AuditRepository

	// CommandHistoryRepository 指令历史
	CommandHistoryRepository() repository.CommandHistoryRepository

	// SettingsRepository 系统设置
	SettingsRepository() repository.SettingsRepository

	// ==================== 生命周期 ====================

	// Migrate 执行数据库迁移
	Migrate() error

	// Close 关闭数据库连接
	Close() error
}

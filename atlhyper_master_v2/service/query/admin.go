// atlhyper_master_v2/service/query/admin.go
// QueryAdmin 实现 — 管理查询（审计日志、命令历史、事件历史、通知渠道、设置、AI Provider）
package query

import (
	"context"

	"AtlHyper/atlhyper_master_v2/database"
)

// ==================== Audit ====================

func (q *QueryService) ListAuditLogs(ctx context.Context, opts database.AuditQueryOpts) ([]*database.AuditLog, error) {
	return q.auditRepo.List(ctx, opts)
}

func (q *QueryService) CountAuditLogs(ctx context.Context, opts database.AuditQueryOpts) (int64, error) {
	return q.auditRepo.Count(ctx, opts)
}

// ==================== Command History ====================

func (q *QueryService) ListCommandHistory(ctx context.Context, opts database.CommandQueryOpts) ([]*database.CommandHistory, error) {
	return q.commandRepo.List(ctx, opts)
}

func (q *QueryService) CountCommandHistory(ctx context.Context, opts database.CommandQueryOpts) (int64, error) {
	return q.commandRepo.Count(ctx, opts)
}

// ==================== Event History ====================

func (q *QueryService) ListEventHistory(ctx context.Context, clusterID string, opts database.EventQueryOpts) ([]*database.ClusterEvent, error) {
	return q.eventRepo.ListByCluster(ctx, clusterID, opts)
}

func (q *QueryService) CountEventHistory(ctx context.Context, clusterID string) (int64, error) {
	return q.eventRepo.CountByCluster(ctx, clusterID)
}

// ==================== Notify ====================

func (q *QueryService) ListNotifyChannels(ctx context.Context) ([]*database.NotifyChannel, error) {
	return q.notifyRepo.List(ctx)
}

func (q *QueryService) GetNotifyChannelByType(ctx context.Context, channelType string) (*database.NotifyChannel, error) {
	return q.notifyRepo.GetByType(ctx, channelType)
}

// ==================== Settings ====================

func (q *QueryService) GetSetting(ctx context.Context, key string) (*database.Setting, error) {
	return q.settingsRepo.Get(ctx, key)
}

// ==================== AI Provider ====================

func (q *QueryService) ListAIProviders(ctx context.Context) ([]*database.AIProvider, error) {
	return q.aiProviderRepo.List(ctx)
}

func (q *QueryService) GetAIProviderByID(ctx context.Context, id int64) (*database.AIProvider, error) {
	return q.aiProviderRepo.GetByID(ctx, id)
}

func (q *QueryService) GetAIActiveConfig(ctx context.Context) (*database.AIActiveConfig, error) {
	return q.aiActiveRepo.Get(ctx)
}

func (q *QueryService) ListAIModels(ctx context.Context) ([]*database.AIProviderModel, error) {
	return q.aiModelRepo.ListAll(ctx)
}

// atlhyper_master_v2/service/query/admin.go
// QueryAdmin 实现 — 管理查询（审计日志、命令历史、事件历史、设置、AI Provider）
package query

import (
	"context"

	"AtlHyper/atlhyper_master_v2/database"
)

// ==================== Audit ====================

func (q *AdminQuery) ListAuditLogs(ctx context.Context, opts database.AuditQueryOpts) ([]*database.AuditLog, error) {
	return q.auditRepo.List(ctx, opts)
}

func (q *AdminQuery) CountAuditLogs(ctx context.Context, opts database.AuditQueryOpts) (int64, error) {
	return q.auditRepo.Count(ctx, opts)
}

// ==================== Command History ====================

func (q *AdminQuery) ListCommandHistory(ctx context.Context, opts database.CommandQueryOpts) ([]*database.CommandHistory, error) {
	return q.commandRepo.List(ctx, opts)
}

func (q *AdminQuery) CountCommandHistory(ctx context.Context, opts database.CommandQueryOpts) (int64, error) {
	return q.commandRepo.Count(ctx, opts)
}

// ==================== Event History ====================

func (q *AdminQuery) ListEventHistory(ctx context.Context, clusterID string, opts database.EventQueryOpts) ([]*database.ClusterEvent, error) {
	return q.eventRepo.ListByCluster(ctx, clusterID, opts)
}

func (q *AdminQuery) CountEventHistory(ctx context.Context, clusterID string) (int64, error) {
	return q.eventRepo.CountByCluster(ctx, clusterID)
}

// ==================== AI Provider ====================

func (q *AdminQuery) ListAIProviders(ctx context.Context) ([]*database.AIProvider, error) {
	return q.aiProviderRepo.List(ctx)
}

func (q *AdminQuery) GetAIProviderByID(ctx context.Context, id int64) (*database.AIProvider, error) {
	return q.aiProviderRepo.GetByID(ctx, id)
}

func (q *AdminQuery) GetAISettings(ctx context.Context) (*database.AISettings, error) {
	return q.aiSettingsRepo.Get(ctx)
}

func (q *AdminQuery) ListAIModels(ctx context.Context) ([]*database.AIProviderModel, error) {
	return q.aiModelRepo.ListAll(ctx)
}

// ==================== AI Role Budget ====================

func (q *AdminQuery) ListAIRoleBudgets(ctx context.Context) ([]*database.AIRoleBudget, error) {
	return q.aiBudgetRepo.ListAll(ctx)
}

// ==================== AI Reports ====================

func (q *AdminQuery) ListRecentAIReports(ctx context.Context, role string, limit, offset int) ([]*database.AIReport, int, error) {
	return q.aiReportRepo.ListRecent(ctx, role, limit, offset)
}

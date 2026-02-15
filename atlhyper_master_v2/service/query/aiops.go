// atlhyper_master_v2/service/query/aiops.go
// AIOps 查询实现
package query

import (
	"context"
	"time"

	"AtlHyper/atlhyper_master_v2/aiops"
)

// GetAIOpsGraph 获取指定集群的依赖图
func (q *QueryService) GetAIOpsGraph(ctx context.Context, clusterID string) (*aiops.DependencyGraph, error) {
	if q.aiopsEngine == nil {
		return nil, nil
	}
	return q.aiopsEngine.GetGraph(clusterID), nil
}

// GetAIOpsGraphTrace 追踪指定实体的上下游链路
func (q *QueryService) GetAIOpsGraphTrace(ctx context.Context, clusterID, fromKey, direction string, maxDepth int) (*aiops.TraceResult, error) {
	if q.aiopsEngine == nil {
		return &aiops.TraceResult{}, nil
	}
	return q.aiopsEngine.GetGraphTrace(clusterID, fromKey, direction, maxDepth), nil
}

// GetAIOpsBaseline 获取指定实体的基线状态
func (q *QueryService) GetAIOpsBaseline(ctx context.Context, clusterID, entityKey string) (*aiops.EntityBaseline, error) {
	if q.aiopsEngine == nil {
		return nil, nil
	}
	return q.aiopsEngine.GetBaseline(entityKey), nil
}

// GetAIOpsClusterRisk 获取集群风险评分
func (q *QueryService) GetAIOpsClusterRisk(ctx context.Context, clusterID string) (*aiops.ClusterRisk, error) {
	if q.aiopsEngine == nil {
		return nil, nil
	}
	return q.aiopsEngine.GetClusterRisk(clusterID), nil
}

// GetAIOpsEntityRisks 获取实体风险列表
func (q *QueryService) GetAIOpsEntityRisks(ctx context.Context, clusterID, sortBy string, limit int) ([]*aiops.EntityRisk, error) {
	if q.aiopsEngine == nil {
		return nil, nil
	}
	return q.aiopsEngine.GetEntityRisks(clusterID, sortBy, limit), nil
}

// GetAIOpsEntityRisk 获取单个实体的风险详情
func (q *QueryService) GetAIOpsEntityRisk(ctx context.Context, clusterID, entityKey string) (*aiops.EntityRiskDetail, error) {
	if q.aiopsEngine == nil {
		return nil, nil
	}
	return q.aiopsEngine.GetEntityRisk(clusterID, entityKey), nil
}

// GetAIOpsIncidents 查询事件列表
func (q *QueryService) GetAIOpsIncidents(ctx context.Context, opts aiops.IncidentQueryOpts) ([]*aiops.Incident, int, error) {
	if q.aiopsEngine == nil {
		return nil, 0, nil
	}
	return q.aiopsEngine.GetIncidents(ctx, opts)
}

// GetAIOpsIncidentDetail 获取事件详情
func (q *QueryService) GetAIOpsIncidentDetail(ctx context.Context, incidentID string) (*aiops.IncidentDetail, error) {
	if q.aiopsEngine == nil {
		return nil, nil
	}
	return q.aiopsEngine.GetIncidentDetail(ctx, incidentID), nil
}

// GetAIOpsIncidentStats 获取事件统计
func (q *QueryService) GetAIOpsIncidentStats(ctx context.Context, clusterID string, since time.Time) (*aiops.IncidentStats, error) {
	if q.aiopsEngine == nil {
		return nil, nil
	}
	return q.aiopsEngine.GetIncidentStats(ctx, clusterID, since), nil
}

// GetAIOpsIncidentPatterns 获取历史事件模式
func (q *QueryService) GetAIOpsIncidentPatterns(ctx context.Context, entityKey string, since time.Time) ([]*aiops.IncidentPattern, error) {
	if q.aiopsEngine == nil {
		return nil, nil
	}
	return q.aiopsEngine.GetIncidentPatterns(ctx, entityKey, since), nil
}

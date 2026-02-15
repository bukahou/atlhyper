// atlhyper_master_v2/service/query/aiops.go
// AIOps 查询实现
package query

import (
	"context"

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

// atlhyper_master_v2/service/query/otel.go
// OTel 快照/时间线查询实现
package query

import (
	"context"
	"time"

	"AtlHyper/model_v3/cluster"
)

// GetOTelSnapshot 从内存快照中读取 OTel 数据
func (q *QueryService) GetOTelSnapshot(ctx context.Context, clusterID string) (*cluster.OTelSnapshot, error) {
	snapshot, err := q.store.GetSnapshot(clusterID)
	if err != nil || snapshot == nil {
		return nil, err
	}
	return snapshot.OTel, nil
}

// GetOTelTimeline 获取 OTel 时间线数据
func (q *QueryService) GetOTelTimeline(ctx context.Context, clusterID string, since time.Time) ([]cluster.OTelEntry, error) {
	return q.store.GetOTelTimeline(clusterID, since)
}

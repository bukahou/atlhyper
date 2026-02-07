// Package sync 同步服务
//
// slo_persist.go - SLO 指标持久化服务
//
// 本文件实现 SLOPersistService，负责:
//   - 从 DataHub 读取 ClusterSnapshot 中的 SLOData
//   - 调用 slo.Processor 写入 SQLite（指标 + 路由映射）
package sync

import (
	"context"

	"AtlHyper/atlhyper_master_v2/datahub"
	"AtlHyper/atlhyper_master_v2/slo"
	"AtlHyper/common/logger"
)

var sloLog = logger.Module("SLOPersist")

// SLOPersistService SLO 指标持久化服务
type SLOPersistService struct {
	store        datahub.Store
	sloProcessor *slo.Processor
}

// NewSLOPersistService 创建 SLO 持久化服务
func NewSLOPersistService(store datahub.Store, sloProcessor *slo.Processor) *SLOPersistService {
	return &SLOPersistService{
		store:        store,
		sloProcessor: sloProcessor,
	}
}

// Sync 同步指定集群的 SLO 数据到 SQLite
// 由 Processor 在收到快照后通过 OnSnapshotReceived 回调触发
func (s *SLOPersistService) Sync(clusterID string) error {
	snapshot, err := s.store.GetSnapshot(clusterID)
	if err != nil {
		return err
	}
	if snapshot == nil || snapshot.SLOData == nil {
		return nil
	}

	ctx := context.Background()

	// 处理 SLO 指标
	if err := s.sloProcessor.ProcessIngressMetrics(ctx, clusterID, &snapshot.SLOData.Metrics); err != nil {
		sloLog.Error("SLO 指标处理失败", "cluster", clusterID, "err", err)
		return err
	}

	// 处理路由映射
	if len(snapshot.SLOData.Routes) > 0 {
		if err := s.sloProcessor.ProcessIngressRoutes(ctx, clusterID, snapshot.SLOData.Routes); err != nil {
			sloLog.Error("SLO 路由映射处理失败", "cluster", clusterID, "err", err)
			// 不返回错误，继续处理
		}
	}

	return nil
}

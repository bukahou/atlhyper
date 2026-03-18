// atlhyper_master_v2/service/operations/slo.go
// SLO 写入服务 — 接收 model 类型请求，转换为 database 类型后写入
package operations

import (
	"context"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/model"
)

// SLOService SLO 写入服务
type SLOService struct {
	sloRepo database.SLORepository
}

// NewSLOService 创建 SLOService
func NewSLOService(sloRepo database.SLORepository) *SLOService {
	return &SLOService{sloRepo: sloRepo}
}

// UpsertSLOTarget 创建或更新 SLO 目标（model → database 转换）
func (s *SLOService) UpsertSLOTarget(ctx context.Context, req *model.UpdateSLOTargetRequest) error {
	target := &database.SLOTarget{
		ClusterID:          req.ClusterID,
		Host:               req.Host,
		TimeRange:          req.TimeRange,
		AvailabilityTarget: req.AvailabilityTarget,
		P95LatencyTarget:   req.P95LatencyTarget,
	}
	return s.sloRepo.UpsertTarget(ctx, target)
}

// Package service 业务逻辑层
//
// slo_service.go - SLO 指标采集服务
//
// 本文件实现 SLOService 接口，负责协调 SLO 数据采集。
// 遵循 Service → Repository → SDK 架构，不直接访问 SDK。
//
// 架构位置:
//
//	Scheduler
//	    ↓ 调用
//	SLOService (本文件) ← 业务逻辑
//	    ↓ 调用
//	SLORepository        ← 数据访问 + 增量计算
//	    ↓ 调用
//	IngressClient (SDK)  ← HTTP 采集
package service

import (
	"context"

	"AtlHyper/atlhyper_agent_v2/repository"
	"AtlHyper/common/logger"
	"AtlHyper/model_v2"
)

var sloLog = logger.Module("SLOService")

// sloService SLO 服务实现
type sloService struct {
	sloRepo repository.SLORepository
}

// NewSLOService 创建 SLO 服务
func NewSLOService(sloRepo repository.SLORepository) SLOService {
	return &sloService{
		sloRepo: sloRepo,
	}
}

// Collect 采集 SLO 指标数据
func (s *sloService) Collect(ctx context.Context) (*model_v2.SLOSnapshot, error) {
	// 采集指标
	snapshot, err := s.sloRepo.Collect(ctx)
	if err != nil {
		sloLog.Warn("SLO 指标采集失败", "err", err)
		return nil, err
	}

	if snapshot == nil {
		return nil, nil
	}

	// 采集路由映射
	routes, err := s.sloRepo.CollectRoutes(ctx)
	if err != nil {
		sloLog.Warn("IngressRoute 采集失败", "err", err)
		// 继续，routes 可以为空
	}

	snapshot.Routes = routes

	sloLog.Debug("SLO 数据采集完成",
		"counters", len(snapshot.Metrics.Counters),
		"histograms", len(snapshot.Metrics.Histograms),
		"routes", len(snapshot.Routes),
	)

	return snapshot, nil
}

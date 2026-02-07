// atlhyper_master_v2/slo/cleaner.go
// SLO 数据清理器
// 定期清理过期数据（三层 raw + hourly + 状态历史）
package slo

import (
	"context"
	"log"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
)

// CleanerConfig 清理器配置
type CleanerConfig struct {
	RawRetention    time.Duration // raw 数据保留时间（默认 48h）
	HourlyRetention time.Duration // hourly 数据保留时间（默认 90d）
	StatusRetention time.Duration // 状态历史保留时间（默认 180d）
	Interval        time.Duration // 清理间隔（默认 1h）
}

// DefaultCleanerConfig 默认清理器配置
func DefaultCleanerConfig() CleanerConfig {
	return CleanerConfig{
		RawRetention:    48 * time.Hour,
		HourlyRetention: 90 * 24 * time.Hour,
		StatusRetention: 180 * 24 * time.Hour,
		Interval:        time.Hour,
	}
}

// Cleaner 数据清理器
type Cleaner struct {
	repo        database.SLORepository
	serviceRepo database.SLOServiceRepository
	edgeRepo    database.SLOEdgeRepository
	cfg         CleanerConfig
	stopCh      chan struct{}
	doneCh      chan struct{}
}

// NewCleaner 创建清理器
func NewCleaner(repo database.SLORepository, serviceRepo database.SLOServiceRepository, edgeRepo database.SLOEdgeRepository, cfg CleanerConfig) *Cleaner {
	return &Cleaner{
		repo:        repo,
		serviceRepo: serviceRepo,
		edgeRepo:    edgeRepo,
		cfg:         cfg,
		stopCh:      make(chan struct{}),
		doneCh:      make(chan struct{}),
	}
}

// Start 启动清理器
func (c *Cleaner) Start() {
	go c.run()
	log.Printf("[SLO Cleaner] 已启动，清理间隔: %v", c.cfg.Interval)
}

// Stop 停止清理器
func (c *Cleaner) Stop() {
	close(c.stopCh)
	<-c.doneCh
	log.Println("[SLO Cleaner] 已停止")
}

func (c *Cleaner) run() {
	defer close(c.doneCh)

	ticker := time.NewTicker(c.cfg.Interval)
	defer ticker.Stop()

	// 启动时立即执行一次
	c.cleanup()

	for {
		select {
		case <-ticker.C:
			c.cleanup()
		case <-c.stopCh:
			return
		}
	}
}

// cleanup 执行一次清理
func (c *Cleaner) cleanup() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	now := time.Now()
	rawBefore := now.Add(-c.cfg.RawRetention)
	hourlyBefore := now.Add(-c.cfg.HourlyRetention)

	// 清理入口 raw 数据
	if count, err := c.repo.DeleteRawMetricsBefore(ctx, rawBefore); err != nil {
		log.Printf("[SLO Cleaner] 清理 ingress raw 失败: %v", err)
	} else if count > 0 {
		log.Printf("[SLO Cleaner] 已清理 %d 条 ingress raw 数据", count)
	}

	// 清理入口 hourly 数据
	if count, err := c.repo.DeleteHourlyMetricsBefore(ctx, hourlyBefore); err != nil {
		log.Printf("[SLO Cleaner] 清理 ingress hourly 失败: %v", err)
	} else if count > 0 {
		log.Printf("[SLO Cleaner] 已清理 %d 条 ingress hourly 数据", count)
	}

	// 清理 service raw 数据
	if count, err := c.serviceRepo.DeleteServiceRawBefore(ctx, rawBefore); err != nil {
		log.Printf("[SLO Cleaner] 清理 service raw 失败: %v", err)
	} else if count > 0 {
		log.Printf("[SLO Cleaner] 已清理 %d 条 service raw 数据", count)
	}

	// 清理 service hourly 数据
	if count, err := c.serviceRepo.DeleteServiceHourlyBefore(ctx, hourlyBefore); err != nil {
		log.Printf("[SLO Cleaner] 清理 service hourly 失败: %v", err)
	} else if count > 0 {
		log.Printf("[SLO Cleaner] 已清理 %d 条 service hourly 数据", count)
	}

	// 清理 edge raw 数据
	if count, err := c.edgeRepo.DeleteEdgeRawBefore(ctx, rawBefore); err != nil {
		log.Printf("[SLO Cleaner] 清理 edge raw 失败: %v", err)
	} else if count > 0 {
		log.Printf("[SLO Cleaner] 已清理 %d 条 edge raw 数据", count)
	}

	// 清理 edge hourly 数据
	if count, err := c.edgeRepo.DeleteEdgeHourlyBefore(ctx, hourlyBefore); err != nil {
		log.Printf("[SLO Cleaner] 清理 edge hourly 失败: %v", err)
	} else if count > 0 {
		log.Printf("[SLO Cleaner] 已清理 %d 条 edge hourly 数据", count)
	}

	// 清理状态历史
	statusBefore := now.Add(-c.cfg.StatusRetention)
	if count, err := c.repo.DeleteStatusHistoryBefore(ctx, statusBefore); err != nil {
		log.Printf("[SLO Cleaner] 清理状态历史失败: %v", err)
	} else if count > 0 {
		log.Printf("[SLO Cleaner] 已清理 %d 条状态历史", count)
	}
}

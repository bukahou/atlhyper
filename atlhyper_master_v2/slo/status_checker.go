// atlhyper_master_v2/slo/status_checker.go
// SLO 状态检测器
// 检测状态变化并记录历史
package slo

import (
	"context"
	"log"
	"sync"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
)

// StatusChecker 状态检测器
type StatusChecker struct {
	repo       database.SLORepository
	mu         sync.Mutex
	lastStatus map[string]string // key: clusterID/host/timeRange -> status
}

// NewStatusChecker 创建状态检测器
func NewStatusChecker(repo database.SLORepository) *StatusChecker {
	return &StatusChecker{
		repo:       repo,
		lastStatus: make(map[string]string),
	}
}

// CheckAndRecord 检查状态变化并记录
func (s *StatusChecker) CheckAndRecord(ctx context.Context, clusterID, host, timeRange string, current *SLOStatus, target *database.SLOTarget) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := clusterID + "/" + host + "/" + timeRange

	// 计算当前状态
	newStatus := DetermineStatus(current.Availability, target.AvailabilityTarget, current.P95Latency, target.P95LatencyTarget)

	// 获取之前的状态
	oldStatus, exists := s.lastStatus[key]
	if !exists {
		oldStatus = "unknown"
	}

	// 如果状态变化，记录历史
	if newStatus != oldStatus {
		history := &database.SLOStatusHistory{
			ClusterID:            clusterID,
			Host:                 host,
			TimeRange:            timeRange,
			OldStatus:            oldStatus,
			NewStatus:            newStatus,
			Availability:         current.Availability,
			P95Latency:           current.P95Latency,
			ErrorBudgetRemaining: current.ErrorBudgetRemaining,
			ChangedAt:            time.Now(),
		}

		if err := s.repo.InsertStatusHistory(ctx, history); err != nil {
			log.Printf("[SLO StatusChecker] 记录状态变更失败: %v", err)
			return err
		}

		log.Printf("[SLO StatusChecker] 状态变更: %s %s -> %s", key, oldStatus, newStatus)
	}

	// 更新缓存
	s.lastStatus[key] = newStatus
	return nil
}

// GetLastStatus 获取最后记录的状态
func (s *StatusChecker) GetLastStatus(clusterID, host, timeRange string) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := clusterID + "/" + host + "/" + timeRange
	status, ok := s.lastStatus[key]
	if !ok {
		return "unknown"
	}
	return status
}

// SLOStatus 当前 SLO 状态
type SLOStatus struct {
	Availability         float64
	P95Latency           int
	ErrorBudgetRemaining float64
}

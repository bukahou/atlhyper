// Package sync 同步服务
//
// metrics_persist.go - 节点指标持久化服务
//
// 本文件实现 MetricsPersistService，负责:
//   - 从 DataHub 读取 ClusterSnapshot 中的 NodeMetrics
//   - 持久化到 SQLite (实时数据 + 历史数据)
//   - 定期清理过期历史数据
package sync

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/datahub"
	"AtlHyper/common/logger"
	"AtlHyper/model_v2"
)

var metricsLog = logger.Module("MetricsPersist")

// MetricsPersistConfig 配置
type MetricsPersistConfig struct {
	SampleInterval  time.Duration // 历史采样间隔 (默认 5 分钟)
	RetentionDays   int           // 历史数据保留天数 (默认 30 天)
	CleanupInterval time.Duration // 清理间隔 (默认 1 小时)
}

// MetricsPersistService 节点指标持久化服务
type MetricsPersistService struct {
	store       datahub.Store
	metricsRepo database.NodeMetricsRepository
	config      MetricsPersistConfig

	// 上次采样时间 (按 clusterID + nodeName)
	lastSample   map[string]time.Time
	lastSampleMu sync.RWMutex

	// 控制
	stopCh chan struct{}
	wg     sync.WaitGroup
}

// NewMetricsPersistService 创建节点指标持久化服务
func NewMetricsPersistService(
	store datahub.Store,
	metricsRepo database.NodeMetricsRepository,
	config MetricsPersistConfig,
) *MetricsPersistService {
	// 默认值
	if config.SampleInterval == 0 {
		config.SampleInterval = 5 * time.Minute
	}
	if config.RetentionDays == 0 {
		config.RetentionDays = 30
	}
	if config.CleanupInterval == 0 {
		config.CleanupInterval = 1 * time.Hour
	}

	return &MetricsPersistService{
		store:       store,
		metricsRepo: metricsRepo,
		config:      config,
		lastSample:  make(map[string]time.Time),
		stopCh:      make(chan struct{}),
	}
}

// Start 启动服务
func (s *MetricsPersistService) Start() error {
	s.wg.Add(1)
	go s.cleanupLoop()
	metricsLog.Info("节点指标持久化服务已启动")
	return nil
}

// Stop 停止服务
func (s *MetricsPersistService) Stop() error {
	close(s.stopCh)
	s.wg.Wait()
	metricsLog.Info("节点指标持久化服务已停止")
	return nil
}

// Sync 同步指定集群的节点指标
// 由 Processor 在收到快照后调用
func (s *MetricsPersistService) Sync(clusterID string) error {
	snapshot, err := s.store.GetSnapshot(clusterID)
	if err != nil {
		return err
	}
	if snapshot == nil || snapshot.NodeMetrics == nil {
		return nil
	}

	ctx := context.Background()

	for nodeName, metrics := range snapshot.NodeMetrics {
		// 1. 持久化实时数据
		if err := s.persistLatest(ctx, clusterID, nodeName, metrics); err != nil {
			metricsLog.Error("持久化实时数据失败", "cluster", clusterID, "node", nodeName, "err", err)
			continue
		}

		// 2. 按间隔采样历史数据
		if s.shouldSample(clusterID, nodeName) {
			if err := s.persistHistory(ctx, clusterID, metrics); err != nil {
				metricsLog.Error("持久化历史数据失败", "cluster", clusterID, "node", nodeName, "err", err)
			}
		}
	}

	return nil
}

// persistLatest 持久化实时数据
func (s *MetricsPersistService) persistLatest(ctx context.Context, clusterID, nodeName string, metrics *model_v2.NodeMetricsSnapshot) error {
	// 序列化完整数据
	jsonData, err := json.Marshal(metrics)
	if err != nil {
		return err
	}

	// 提取主磁盘使用率
	diskUsage := 0.0
	if disk := metrics.GetPrimaryDisk(); disk != nil {
		diskUsage = disk.UsagePercent
	}

	latest := &database.NodeMetricsLatest{
		ClusterID:    clusterID,
		NodeName:     nodeName,
		SnapshotJSON: string(jsonData),
		CPUUsage:     metrics.CPU.UsagePercent,
		MemoryUsage:  metrics.Memory.UsagePercent,
		DiskUsage:    diskUsage,
		CPUTemp:      metrics.Temperature.CPUTemp,
		UpdatedAt:    time.Now(),
	}

	return s.metricsRepo.UpsertLatest(ctx, latest)
}

// persistHistory 持久化历史数据
func (s *MetricsPersistService) persistHistory(ctx context.Context, clusterID string, metrics *model_v2.NodeMetricsSnapshot) error {
	dp := metrics.ToDataPoint()

	// 调试日志
	metricsLog.Debug("持久化历史数据",
		"node", metrics.NodeName,
		"disks_count", len(metrics.Disks),
		"disk_usage", dp.DiskUsage,
		"cpu_temp", dp.CPUTemp,
	)

	history := &database.NodeMetricsHistory{
		ClusterID:   clusterID,
		NodeName:    metrics.NodeName,
		Timestamp:   dp.Timestamp,
		CPUUsage:    dp.CPUUsage,
		MemoryUsage: dp.MemoryUsage,
		DiskUsage:   dp.DiskUsage,
		DiskIORead:  dp.DiskIORead,
		DiskIOWrite: dp.DiskIOWrite,
		NetworkRx:   dp.NetworkRx,
		NetworkTx:   dp.NetworkTx,
		CPUTemp:     dp.CPUTemp,
		Load1:       dp.Load1,
	}

	return s.metricsRepo.InsertHistory(ctx, history)
}

// shouldSample 判断是否应该采样历史数据
func (s *MetricsPersistService) shouldSample(clusterID, nodeName string) bool {
	key := clusterID + "/" + nodeName

	s.lastSampleMu.RLock()
	lastTime, ok := s.lastSample[key]
	s.lastSampleMu.RUnlock()

	if !ok || time.Since(lastTime) >= s.config.SampleInterval {
		s.lastSampleMu.Lock()
		s.lastSample[key] = time.Now()
		s.lastSampleMu.Unlock()
		return true
	}
	return false
}

// cleanupLoop 清理循环
func (s *MetricsPersistService) cleanupLoop() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.cleanup()
		}
	}
}

// cleanup 清理过期数据
func (s *MetricsPersistService) cleanup() {
	ctx := context.Background()
	before := time.Now().AddDate(0, 0, -s.config.RetentionDays)

	deleted, err := s.metricsRepo.DeleteHistoryBefore(ctx, before)
	if err != nil {
		metricsLog.Error("清理历史数据失败", "err", err)
		return
	}

	if deleted > 0 {
		metricsLog.Info("已清理过期历史数据", "count", deleted)
	}
}

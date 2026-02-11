package metrics

import (
	"context"
	"sync"
	"time"

	"AtlHyper/atlhyper_agent_v2/repository"
	"AtlHyper/atlhyper_agent_v2/sdk"
	"AtlHyper/common/logger"
	"AtlHyper/model_v2"
)

var log = logger.Module("MetricsRepo")

// metricsRepository 节点指标仓库实现
//
// OTel 模式: Sync() 从 OTelClient 拉取 node_exporter 原始数据，
// 计算 counter rate 后转换为 NodeMetricsSnapshot。
// Receiver 模式: 降级到 ReceiverClient 被动接收数据。
type metricsRepository struct {
	otel     sdk.OTelClient
	receiver sdk.ReceiverClient // 降级用

	mu       sync.RWMutex
	prev     map[string]*sdk.OTelNodeRawMetrics
	prevTime time.Time
	current  map[string]*model_v2.NodeMetricsSnapshot
}

// NewMetricsRepository 创建 OTel 模式的节点指标仓库
//
// otel: OTel Collector 客户端（主要数据源）
// receiver: ReceiverClient（降级数据源，可为 nil）
func NewMetricsRepository(otel sdk.OTelClient, receiver sdk.ReceiverClient) repository.MetricsRepository {
	return &metricsRepository{
		otel:     otel,
		receiver: receiver,
	}
}

// NewLegacyMetricsRepository 创建旧模式的节点指标仓库（仅 Receiver）
func NewLegacyMetricsRepository(receiver sdk.ReceiverClient) repository.MetricsRepository {
	return &metricsRepository{
		receiver: receiver,
	}
}

// GetAll 获取所有节点的最新指标快照
func (r *metricsRepository) GetAll() map[string]*model_v2.NodeMetricsSnapshot {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.current != nil {
		return r.current
	}

	// OTel 尚未产出数据，降级到 Receiver
	if r.receiver != nil {
		return r.receiver.GetAllNodeMetrics()
	}
	return nil
}

// Sync 从 OTel Collector 拉取最新节点指标
//
// 流程:
//  1. 调用 OTelClient.ScrapeNodeMetrics() 拉取原始数据
//  2. 失败时降级到 ReceiverClient
//  3. 成功时: 如有上次数据 (prev), 计算 rate 并转换为 NodeMetricsSnapshot
//  4. 首次 Sync 只存原始值，不输出快照（需要两次采样）
func (r *metricsRepository) Sync(ctx context.Context) error {
	if r.otel == nil {
		return nil // 无 OTel 客户端，使用 Receiver 模式
	}

	raw, err := r.otel.ScrapeNodeMetrics(ctx)
	if err != nil {
		log.Warn("OTel 节点指标采集失败，降级到 Receiver", "err", err)
		return nil // 降级，不返回错误
	}

	now := time.Now()

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.prev == nil {
		// 首次采集: 只存原始值
		r.prev = raw
		r.prevTime = now
		log.Info("首次节点指标采集完成，等待下次采集计算 rate", "nodes", len(raw))
		return nil
	}

	// 计算 elapsed
	elapsed := now.Sub(r.prevTime).Seconds()
	if elapsed <= 0 {
		elapsed = 15.0 // 默认间隔
	}

	// 转换为 NodeMetricsSnapshot
	snapshots := make(map[string]*model_v2.NodeMetricsSnapshot, len(raw))
	for nodeName, cur := range raw {
		prev := r.prev[nodeName]
		snap := convertToSnapshot(nodeName, cur, prev, elapsed)
		snapshots[nodeName] = snap
	}

	r.current = snapshots
	r.prev = raw
	r.prevTime = now

	log.Debug("节点指标已更新", "nodes", len(snapshots))
	return nil
}

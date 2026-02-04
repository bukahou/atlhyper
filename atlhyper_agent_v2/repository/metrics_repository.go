// Package repository 数据访问层
//
// metrics_repository.go - 节点指标数据仓库
//
// 本文件实现 MetricsRepository，用于:
//   - 接收 atlhyper_metrics_v2 推送的节点指标
//   - 存储在内存中供 SnapshotService 聚合
//   - 按节点名覆盖式存储（只保留最新数据）
package repository

import (
	"sync"

	"AtlHyper/model_v2"
)

// MetricsRepository 节点指标仓库接口
type MetricsRepository interface {
	// Save 保存节点指标（覆盖式）
	Save(snapshot *model_v2.NodeMetricsSnapshot)
	// Get 获取指定节点的指标
	Get(nodeName string) *model_v2.NodeMetricsSnapshot
	// GetAll 获取所有节点的指标
	GetAll() map[string]*model_v2.NodeMetricsSnapshot
	// Count 返回节点数量
	Count() int
	// Delete 删除指定节点的指标
	Delete(nodeName string)
}

// metricsRepository 节点指标仓库实现
type metricsRepository struct {
	mu   sync.RWMutex
	data map[string]*model_v2.NodeMetricsSnapshot
}

// NewMetricsRepository 创建节点指标仓库
func NewMetricsRepository() MetricsRepository {
	return &metricsRepository{
		data: make(map[string]*model_v2.NodeMetricsSnapshot),
	}
}

// Save 保存节点指标
// 按 NodeName 覆盖式存储
func (r *metricsRepository) Save(snapshot *model_v2.NodeMetricsSnapshot) {
	if snapshot == nil || snapshot.NodeName == "" {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.data[snapshot.NodeName] = snapshot
}

// Get 获取指定节点的指标
func (r *metricsRepository) Get(nodeName string) *model_v2.NodeMetricsSnapshot {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.data[nodeName]
}

// GetAll 获取所有节点的指标
// 返回的是数据的副本，避免外部修改
func (r *metricsRepository) GetAll() map[string]*model_v2.NodeMetricsSnapshot {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// 返回副本
	result := make(map[string]*model_v2.NodeMetricsSnapshot, len(r.data))
	for k, v := range r.data {
		result[k] = v
	}
	return result
}

// Count 返回节点数量
func (r *metricsRepository) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.data)
}

// Delete 删除指定节点的指标
func (r *metricsRepository) Delete(nodeName string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.data, nodeName)
}

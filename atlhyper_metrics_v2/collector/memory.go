package collector

import (
	"sync"

	"AtlHyper/atlhyper_metrics_v2/config"
	"AtlHyper/atlhyper_metrics_v2/utils"
	"AtlHyper/model_v2"
)

// memoryCollector 内存采集器实现
type memoryCollector struct {
	cfg *config.Config

	mu      sync.RWMutex
	metrics model_v2.MemoryMetrics
}

// NewMemoryCollector 创建内存采集器
func NewMemoryCollector(cfg *config.Config) MemoryCollector {
	return &memoryCollector{
		cfg: cfg,
	}
}

// Collect 采集内存指标
func (c *memoryCollector) Collect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	path := c.cfg.Paths.ProcRoot + "/meminfo"
	lines, err := utils.ReadFileLines(path)
	if err != nil {
		return err
	}

	memInfo := make(map[string]int64)
	for _, line := range lines {
		key, value := utils.ParseKeyValue(line)
		if key != "" {
			memInfo[key] = utils.ParseMemValue(value)
		}
	}

	// 物理内存
	c.metrics.Total = memInfo["MemTotal"]
	c.metrics.Free = memInfo["MemFree"]
	c.metrics.Available = memInfo["MemAvailable"]
	c.metrics.Cached = memInfo["Cached"]
	c.metrics.Buffers = memInfo["Buffers"]

	// 计算已用内存
	// Used = Total - Available (更准确的计算方式)
	if c.metrics.Available > 0 {
		c.metrics.Used = c.metrics.Total - c.metrics.Available
	} else {
		// 回退计算: Used = Total - Free - Cached - Buffers
		c.metrics.Used = c.metrics.Total - c.metrics.Free - c.metrics.Cached - c.metrics.Buffers
	}

	// 使用率
	if c.metrics.Total > 0 {
		c.metrics.UsagePercent = float64(c.metrics.Used) / float64(c.metrics.Total) * 100
	}

	// Swap
	c.metrics.SwapTotal = memInfo["SwapTotal"]
	c.metrics.SwapFree = memInfo["SwapFree"]
	c.metrics.SwapUsed = c.metrics.SwapTotal - c.metrics.SwapFree

	if c.metrics.SwapTotal > 0 {
		c.metrics.SwapPercent = float64(c.metrics.SwapUsed) / float64(c.metrics.SwapTotal) * 100
	}

	return nil
}

// Get 获取内存指标
func (c *memoryCollector) Get() model_v2.MemoryMetrics {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.metrics
}

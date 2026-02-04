// Package collector 指标采集器
package collector

import "AtlHyper/model_v2"

// Collector 采集器通用接口
type Collector interface {
	// Collect 执行一次采集
	Collect() error
}

// CPUCollector CPU 采集器接口
type CPUCollector interface {
	Collector
	// Get 获取 CPU 指标
	Get() model_v2.CPUMetrics
	// Start 启动后台采样
	Start()
	// Stop 停止后台采样
	Stop()
}

// MemoryCollector 内存采集器接口
type MemoryCollector interface {
	Collector
	// Get 获取内存指标
	Get() model_v2.MemoryMetrics
}

// DiskCollector 磁盘采集器接口
type DiskCollector interface {
	Collector
	// Get 获取磁盘指标列表
	Get() []model_v2.DiskMetrics
	// Start 启动后台采样
	Start()
	// Stop 停止后台采样
	Stop()
}

// NetworkCollector 网络采集器接口
type NetworkCollector interface {
	Collector
	// Get 获取网络指标列表
	Get() []model_v2.NetworkMetrics
	// Start 启动后台采样
	Start()
	// Stop 停止后台采样
	Stop()
}

// TemperatureCollector 温度采集器接口
type TemperatureCollector interface {
	Collector
	// Get 获取温度指标
	Get() model_v2.TemperatureMetrics
}

// ProcessCollector 进程采集器接口
type ProcessCollector interface {
	Collector
	// Get 获取 Top N 进程指标
	Get() []model_v2.ProcessMetrics
	// Start 启动后台采样
	Start()
	// Stop 停止后台采样
	Stop()
}

package internal

import (
	"log"
	"os"
	"time"

	"NeuroController/metrics/collect"
	"NeuroController/model/metrics"
)

func BuildNodeMetricsSnapshot() *metrics.NodeMetricsSnapshot {
	var (
		diskStats    []metrics.DiskStat
		networkStats []metrics.NetworkStat
		tempStat     metrics.TemperatureStat
		cpuStat      metrics.CPUStat
		memStat      metrics.MemoryStat
		topList      []metrics.TopCPUProcess
	)

	// 获取宿主机名称
	hostname := os.Getenv("NODE_NAME")
	if hostname == "" {
		hostname, _ = os.Hostname()
	}
	// CPU & Top 进程
	if cs, tl, err := collect.CollectCPU(); err == nil {
		cpuStat = cs
		topList = tl
	} else {
		log.Printf("❌ [Metrics] CPU采集失败: %v", err)
	}

	// 内存
	if ms, err := collect.CollectMemory(); err == nil {
		memStat = ms
	} else {
		log.Printf("❌ [Metrics] 内存采集失败: %v", err)
	}

	// 磁盘
	if ds, err := collect.CollectDisk(); err == nil {
		diskStats = ds
	} else {
		log.Printf("❌ [Metrics] 磁盘采集失败: %v", err)
	}

	// 网络
	if ns, err := collect.CollectNetwork(); err == nil {
		networkStats = ns
	} else {
		log.Printf("❌ [Metrics] 网络采集失败: %v", err)
	}

	// 温度
	if ts, err := collect.CollectTemperature(); err == nil {
		tempStat = ts
	} else {
		log.Printf("❌ [Metrics] 温度采集失败: %v", err)
	}

	// ✅ 全量聚合结构返回
	return &metrics.NodeMetricsSnapshot{
		NodeName:        hostname,
		Timestamp:       time.Now(),
		CPU:             cpuStat,
		Memory:          memStat,
		Temperature:     tempStat,
		Disk:            diskStats,
		Network:         networkStats,
		TopCPUProcesses: topList,
	}
}

// Package aggregator 指标聚合
package aggregator

import (
	"os"
	"strings"
	"time"

	"AtlHyper/atlhyper_metrics_v2/collector"
	"AtlHyper/atlhyper_metrics_v2/config"
	"AtlHyper/atlhyper_metrics_v2/utils"
	"AtlHyper/model_v2"
)

// Aggregator 指标聚合器
type Aggregator struct {
	cfg *config.Config

	cpu         collector.CPUCollector
	memory      collector.MemoryCollector
	disk        collector.DiskCollector
	network     collector.NetworkCollector
	temperature collector.TemperatureCollector
	process     collector.ProcessCollector
}

// New 创建聚合器
func New(cfg *config.Config) *Aggregator {
	return &Aggregator{
		cfg:         cfg,
		cpu:         collector.NewCPUCollector(cfg),
		memory:      collector.NewMemoryCollector(cfg),
		disk:        collector.NewDiskCollector(cfg),
		network:     collector.NewNetworkCollector(cfg),
		temperature: collector.NewTemperatureCollector(cfg),
		process:     collector.NewProcessCollector(cfg),
	}
}

// Start 启动所有采集器的后台采样
func (a *Aggregator) Start() {
	a.cpu.Start()
	a.disk.Start()
	a.network.Start()
	a.process.Start()
}

// Stop 停止所有采集器
func (a *Aggregator) Stop() {
	a.cpu.Stop()
	a.disk.Stop()
	a.network.Stop()
	a.process.Stop()
}

// Collect 执行一次完整采集并返回快照
func (a *Aggregator) Collect() (*model_v2.NodeMetricsSnapshot, error) {
	// 触发所有采集器采集
	a.cpu.Collect()
	a.memory.Collect()
	a.disk.Collect()
	a.network.Collect()
	a.temperature.Collect()
	a.process.Collect()

	// 构建快照
	snapshot := &model_v2.NodeMetricsSnapshot{
		NodeName:    a.cfg.NodeName,
		Timestamp:   time.Now(),
		Hostname:    a.cfg.Hostname,
		OS:          a.getOS(),
		Kernel:      a.getKernel(),
		Uptime:      a.getUptime(),
		CPU:         a.cpu.Get(),
		Memory:      a.memory.Get(),
		Disks:       a.disk.Get(),
		Networks:    a.network.Get(),
		Temperature: a.temperature.Get(),
		Processes:   a.process.Get(),
	}

	return snapshot, nil
}

// getOS 获取操作系统信息
func (a *Aggregator) getOS() string {
	// 尝试从 /etc/os-release 读取
	path := a.cfg.Paths.HostRoot + "/etc/os-release"
	if a.cfg.Paths.HostRoot == "/" {
		path = "/etc/os-release"
	}

	lines, err := utils.ReadFileLines(path)
	if err != nil {
		return "Linux"
	}

	for _, line := range lines {
		if strings.HasPrefix(line, "PRETTY_NAME=") {
			value := strings.TrimPrefix(line, "PRETTY_NAME=")
			value = strings.Trim(value, "\"")
			return value
		}
	}

	return "Linux"
}

// getKernel 获取内核版本
func (a *Aggregator) getKernel() string {
	// 从 /proc/version 或 uname -r
	path := a.cfg.Paths.ProcRoot + "/version"
	content, err := utils.ReadFileString(path)
	if err != nil {
		return ""
	}

	// 格式: Linux version 5.15.0-generic (...)
	fields := strings.Fields(content)
	if len(fields) >= 3 {
		return fields[2]
	}

	return ""
}

// getUptime 获取运行时长（秒）
func (a *Aggregator) getUptime() int64 {
	path := a.cfg.Paths.ProcRoot + "/uptime"
	content, err := utils.ReadFileString(path)
	if err != nil {
		return 0
	}

	// 格式: 12345.67 67890.12
	fields := strings.Fields(content)
	if len(fields) >= 1 {
		// 解析为浮点数再转整数
		var uptime float64
		if _, err := os.Open(path); err == nil {
			// 重新解析
			fields := strings.Fields(content)
			if len(fields) >= 1 {
				if u, err := parseFloat(fields[0]); err == nil {
					uptime = u
				}
			}
		}
		return int64(uptime)
	}

	return 0
}

func parseFloat(s string) (float64, error) {
	var f float64
	for i, c := range s {
		if c == '.' {
			// 整数部分
			for _, d := range s[:i] {
				f = f*10 + float64(d-'0')
			}
			// 小数部分
			div := 1.0
			for _, d := range s[i+1:] {
				div *= 10
				f += float64(d-'0') / div
			}
			return f, nil
		}
	}
	// 无小数点
	for _, d := range s {
		if d < '0' || d > '9' {
			break
		}
		f = f*10 + float64(d-'0')
	}
	return f, nil
}

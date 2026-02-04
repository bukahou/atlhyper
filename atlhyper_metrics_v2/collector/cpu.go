package collector

import (
	"bufio"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"AtlHyper/atlhyper_metrics_v2/config"
	"AtlHyper/atlhyper_metrics_v2/utils"
	"AtlHyper/model_v2"
)

// cpuCollector CPU 采集器实现
type cpuCollector struct {
	cfg *config.Config

	mu      sync.RWMutex
	metrics model_v2.CPUMetrics

	// 采样数据
	prevSample *utils.CPURawSample
	currSample *utils.CPURawSample

	// 控制
	stopCh chan struct{}
	wg     sync.WaitGroup
}

// NewCPUCollector 创建 CPU 采集器
func NewCPUCollector(cfg *config.Config) CPUCollector {
	return &cpuCollector{
		cfg:    cfg,
		stopCh: make(chan struct{}),
	}
}

// Start 启动后台采样
func (c *cpuCollector) Start() {
	// 初始采样
	c.sample()

	c.wg.Add(1)
	go c.sampleLoop()
}

// Stop 停止后台采样
func (c *cpuCollector) Stop() {
	close(c.stopCh)
	c.wg.Wait()
}

// sampleLoop 采样循环
func (c *cpuCollector) sampleLoop() {
	defer c.wg.Done()

	ticker := time.NewTicker(c.cfg.Collect.CPUInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopCh:
			return
		case <-ticker.C:
			c.sample()
		}
	}
}

// sample 执行一次采样
func (c *cpuCollector) sample() {
	sample, err := c.readCPUStats()
	if err != nil {
		return
	}

	c.mu.Lock()
	c.prevSample = c.currSample
	c.currSample = sample
	c.mu.Unlock()
}

// Collect 采集 CPU 指标
func (c *cpuCollector) Collect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 需要两次采样才能计算差值
	if c.prevSample == nil || c.currSample == nil {
		return nil
	}

	// 计算使用率
	c.calculateUsage()

	// 获取负载
	c.readLoadAvg()

	// 获取 CPU 信息（只需要一次）
	if c.metrics.Model == "" {
		c.readCPUInfo()
	}

	return nil
}

// Get 获取 CPU 指标
func (c *cpuCollector) Get() model_v2.CPUMetrics {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.metrics
}

// calculateUsage 计算 CPU 使用率
func (c *cpuCollector) calculateUsage() {
	prev := c.prevSample.Total
	curr := c.currSample.Total

	totalDelta := float64(curr.Total() - prev.Total())
	if totalDelta <= 0 {
		return
	}

	c.metrics.UserPercent = utils.Clamp(float64(curr.User-prev.User+curr.Nice-prev.Nice)/totalDelta*100, 0, 100)
	c.metrics.SystemPercent = utils.Clamp(float64(curr.System-prev.System)/totalDelta*100, 0, 100)
	c.metrics.IdlePercent = utils.Clamp(float64(curr.Idle-prev.Idle)/totalDelta*100, 0, 100)
	c.metrics.IOWaitPercent = utils.Clamp(float64(curr.IOWait-prev.IOWait)/totalDelta*100, 0, 100)
	c.metrics.UsagePercent = utils.Clamp(100-c.metrics.IdlePercent, 0, 100)

	// 每核使用率
	c.metrics.PerCore = make([]float64, len(c.currSample.PerCore))
	for i := range c.currSample.PerCore {
		if i >= len(c.prevSample.PerCore) {
			break
		}
		prevCore := c.prevSample.PerCore[i]
		currCore := c.currSample.PerCore[i]
		coreDelta := float64(currCore.Total() - prevCore.Total())
		if coreDelta > 0 {
			idleDelta := float64(currCore.Idle - prevCore.Idle)
			c.metrics.PerCore[i] = utils.Clamp(100-idleDelta/coreDelta*100, 0, 100)
		}
	}
}

// readCPUStats 读取 /proc/stat
func (c *cpuCollector) readCPUStats() (*utils.CPURawSample, error) {
	path := c.cfg.Paths.ProcRoot + "/stat"
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	sample := &utils.CPURawSample{
		Timestamp: time.Now(),
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "cpu ") {
			// 总计行
			sample.Total = parseCPULine(line)
		} else if strings.HasPrefix(line, "cpu") {
			// 单核行 (cpu0, cpu1, ...)
			sample.PerCore = append(sample.PerCore, parseCPULine(line))
		}
	}

	return sample, scanner.Err()
}

// parseCPULine 解析 CPU 行
// 格式: cpu  user nice system idle iowait irq softirq steal guest guest_nice
func parseCPULine(line string) utils.CPURawStats {
	fields := strings.Fields(line)
	if len(fields) < 8 {
		return utils.CPURawStats{}
	}

	parseUint := func(s string) uint64 {
		v, _ := strconv.ParseUint(s, 10, 64)
		return v
	}

	stats := utils.CPURawStats{
		User:    parseUint(fields[1]),
		Nice:    parseUint(fields[2]),
		System:  parseUint(fields[3]),
		Idle:    parseUint(fields[4]),
		IOWait:  parseUint(fields[5]),
		IRQ:     parseUint(fields[6]),
		SoftIRQ: parseUint(fields[7]),
	}

	if len(fields) > 8 {
		stats.Steal = parseUint(fields[8])
	}

	return stats
}

// readLoadAvg 读取 /proc/loadavg
func (c *cpuCollector) readLoadAvg() {
	path := c.cfg.Paths.ProcRoot + "/loadavg"
	content, err := utils.ReadFileString(path)
	if err != nil {
		return
	}

	fields := strings.Fields(content)
	if len(fields) >= 3 {
		c.metrics.Load1, _ = strconv.ParseFloat(fields[0], 64)
		c.metrics.Load5, _ = strconv.ParseFloat(fields[1], 64)
		c.metrics.Load15, _ = strconv.ParseFloat(fields[2], 64)
	}
}

// readCPUInfo 读取 /proc/cpuinfo
func (c *cpuCollector) readCPUInfo() {
	path := c.cfg.Paths.ProcRoot + "/cpuinfo"
	lines, err := utils.ReadFileLines(path)
	if err != nil {
		return
	}

	// 用于计算物理核心数
	physicalIDs := make(map[string]bool)    // 物理 CPU ID 集合
	coreIDs := make(map[string]map[string]bool) // physical_id -> core_id 集合
	coresPerCPU := 0                          // 每个物理 CPU 的核心数（从 cpu cores 字段读取）

	var currentPhysicalID string

	for _, line := range lines {
		key, value := utils.ParseKeyValue(line)
		switch key {
		case "model name":
			if c.metrics.Model == "" {
				c.metrics.Model = value
			}
		case "cpu MHz":
			if c.metrics.Frequency == 0 {
				c.metrics.Frequency, _ = strconv.ParseFloat(value, 64)
			}
		case "physical id":
			currentPhysicalID = value
			physicalIDs[value] = true
			if coreIDs[value] == nil {
				coreIDs[value] = make(map[string]bool)
			}
		case "core id":
			if currentPhysicalID != "" {
				coreIDs[currentPhysicalID][value] = true
			}
		case "cpu cores":
			if cores, err := strconv.Atoi(value); err == nil && cores > coresPerCPU {
				coresPerCPU = cores
			}
		}
	}

	// 计算物理核心数
	// 方法1: 从 cpu cores 字段直接读取（最可靠）
	// 方法2: 统计唯一的 (physical_id, core_id) 组合
	if coresPerCPU > 0 && len(physicalIDs) > 0 {
		// 物理核心数 = 物理 CPU 数 × 每 CPU 核心数
		c.metrics.Cores = len(physicalIDs) * coresPerCPU
	} else {
		// 回退: 统计唯一核心
		totalCores := 0
		for _, cores := range coreIDs {
			totalCores += len(cores)
		}
		if totalCores > 0 {
			c.metrics.Cores = totalCores
		} else {
			// 最后回退: 假设无超线程
			c.metrics.Cores = runtime.NumCPU()
		}
	}

	// 逻辑线程数 = 系统可见的 CPU 数
	c.metrics.Threads = runtime.NumCPU()
}

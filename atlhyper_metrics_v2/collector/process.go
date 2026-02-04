package collector

import (
	"bufio"
	"os"
	"os/user"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"AtlHyper/atlhyper_metrics_v2/config"
	"AtlHyper/atlhyper_metrics_v2/utils"
	"AtlHyper/model_v2"
)

// processCollector 进程采集器实现
type processCollector struct {
	cfg *config.Config

	mu      sync.RWMutex
	metrics []model_v2.ProcessMetrics

	// 采样数据
	prevSample *utils.ProcRawSample
	currSample *utils.ProcRawSample

	// 用户名缓存
	userCache   map[int]string
	userCacheMu sync.RWMutex

	// 系统信息
	pageSize    int64
	clockTicks  int64
	totalMemory int64

	// 控制
	stopCh chan struct{}
	wg     sync.WaitGroup
}

// NewProcessCollector 创建进程采集器
func NewProcessCollector(cfg *config.Config) ProcessCollector {
	return &processCollector{
		cfg:        cfg,
		userCache:  make(map[int]string),
		pageSize:   4096,           // 默认页大小
		clockTicks: 100,            // 默认 jiffies/秒
		stopCh:     make(chan struct{}),
	}
}

// Start 启动后台采样
func (c *processCollector) Start() {
	// 获取系统信息
	c.getSystemInfo()

	// 初始采样
	c.sample()

	c.wg.Add(1)
	go c.sampleLoop()
}

// Stop 停止后台采样
func (c *processCollector) Stop() {
	close(c.stopCh)
	c.wg.Wait()
}

// sampleLoop 采样循环
func (c *processCollector) sampleLoop() {
	defer c.wg.Done()

	ticker := time.NewTicker(c.cfg.Collect.ProcInterval)
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
func (c *processCollector) sample() {
	sample, err := c.readProcStats()
	if err != nil {
		return
	}

	c.mu.Lock()
	c.prevSample = c.currSample
	c.currSample = sample
	c.mu.Unlock()
}

// Collect 采集进程指标
func (c *processCollector) Collect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.prevSample == nil || c.currSample == nil {
		return nil
	}

	// 获取总内存
	c.getTotalMemory()

	// 计算每个进程的 CPU 使用率
	type procWithCPU struct {
		pid        int
		cpuPercent float64
	}

	var procs []procWithCPU
	elapsed := c.currSample.Timestamp.Sub(c.prevSample.Timestamp).Seconds()
	cpuTimeDelta := float64(c.currSample.TotalCPU - c.prevSample.TotalCPU)

	for pid, curr := range c.currSample.Stats {
		prev, ok := c.prevSample.Stats[pid]
		if !ok {
			continue
		}

		// CPU 使用率 = (进程时间差 / 总 CPU 时间差) * 100
		procTimeDelta := float64((curr.UTime + curr.STime) - (prev.UTime + prev.STime))
		cpuPercent := 0.0
		if cpuTimeDelta > 0 && elapsed > 0 {
			cpuPercent = (procTimeDelta / cpuTimeDelta) * 100
		}

		procs = append(procs, procWithCPU{
			pid:        pid,
			cpuPercent: cpuPercent,
		})
	}

	// 按 CPU 使用率排序
	sort.Slice(procs, func(i, j int) bool {
		return procs[i].cpuPercent > procs[j].cpuPercent
	})

	// 取 Top N
	topN := c.cfg.Collect.TopProcesses
	if topN > len(procs) {
		topN = len(procs)
	}

	c.metrics = make([]model_v2.ProcessMetrics, 0, topN)

	for i := 0; i < topN; i++ {
		pid := procs[i].pid
		proc := c.buildProcessMetrics(pid, procs[i].cpuPercent)
		if proc != nil {
			c.metrics = append(c.metrics, *proc)
		}
	}

	return nil
}

// Get 获取进程指标
func (c *processCollector) Get() []model_v2.ProcessMetrics {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.metrics
}

// getSystemInfo 获取系统信息
func (c *processCollector) getSystemInfo() {
	// 页大小
	c.pageSize = int64(os.Getpagesize())

	// 时钟频率 (jiffies/秒)
	// Linux 默认为 100，但可以是 250/300/1000
	c.clockTicks = 100
}

// getTotalMemory 获取总内存
func (c *processCollector) getTotalMemory() {
	if c.totalMemory > 0 {
		return
	}

	path := c.cfg.Paths.ProcRoot + "/meminfo"
	lines, err := utils.ReadFileLines(path)
	if err != nil {
		return
	}

	for _, line := range lines {
		key, value := utils.ParseKeyValue(line)
		if key == "MemTotal" {
			c.totalMemory = utils.ParseMemValue(value)
			break
		}
	}
}

// readProcStats 读取所有进程统计
func (c *processCollector) readProcStats() (*utils.ProcRawSample, error) {
	procPath := c.cfg.Paths.ProcRoot
	pids, err := utils.ListNumericDirs(procPath)
	if err != nil {
		return nil, err
	}

	sample := &utils.ProcRawSample{
		Timestamp: time.Now(),
		Stats:     make(map[int]utils.ProcRawStats),
	}

	for _, pid := range pids {
		stats, err := c.readProcStat(pid)
		if err != nil {
			continue
		}

		// 过滤线程 (Tgid != PID)
		if stats.PID != pid {
			continue
		}

		sample.Stats[pid] = stats
	}

	// 读取总 CPU 时间
	sample.TotalCPU = c.readTotalCPU()

	return sample, nil
}

// readProcStat 读取单个进程的 /proc/[pid]/stat
func (c *processCollector) readProcStat(pid int) (utils.ProcRawStats, error) {
	path := c.cfg.Paths.ProcRoot + "/" + strconv.Itoa(pid) + "/stat"
	content, err := utils.ReadFileString(path)
	if err != nil {
		return utils.ProcRawStats{}, err
	}

	// 解析 stat 文件
	// 格式: pid (comm) state ppid pgrp session tty_nr tpgid flags minflt cminflt majflt cmajflt utime stime ...
	// comm 可能包含空格和括号，需要特殊处理

	// 找到 comm 的开始和结束括号
	start := strings.Index(content, "(")
	end := strings.LastIndex(content, ")")
	if start == -1 || end == -1 || end <= start {
		return utils.ProcRawStats{}, &parseError{"invalid stat format"}
	}

	name := content[start+1 : end]
	fields := strings.Fields(content[end+2:]) // 跳过 ") "

	if len(fields) < 20 {
		return utils.ProcRawStats{}, &parseError{"insufficient fields"}
	}

	return utils.ProcRawStats{
		PID:       pid,
		Name:      name,
		State:     fields[0][0],
		UTime:     parseUint64(fields[11]),  // utime
		STime:     parseUint64(fields[12]),  // stime
		StartTime: parseUint64(fields[19]),  // starttime
	}, nil
}

// readTotalCPU 读取总 CPU 时间
func (c *processCollector) readTotalCPU() uint64 {
	path := c.cfg.Paths.ProcRoot + "/stat"
	file, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "cpu ") {
			stats := parseCPULine(line)
			return stats.Total()
		}
	}
	return 0
}

// buildProcessMetrics 构建进程指标
func (c *processCollector) buildProcessMetrics(pid int, cpuPercent float64) *model_v2.ProcessMetrics {
	procPath := c.cfg.Paths.ProcRoot + "/" + strconv.Itoa(pid)

	proc := &model_v2.ProcessMetrics{
		PID:        pid,
		CPUPercent: utils.Clamp(cpuPercent, 0, 100),
	}

	// 从 currSample 获取名称
	if stats, ok := c.currSample.Stats[pid]; ok {
		proc.Name = stats.Name
		proc.Status = string(stats.State)
	}

	// 读取 cmdline
	if cmdline, err := utils.ReadFileString(procPath + "/cmdline"); err == nil {
		// cmdline 用 \0 分隔参数
		proc.Cmdline = strings.ReplaceAll(cmdline, "\x00", " ")
		proc.Cmdline = strings.TrimSpace(proc.Cmdline)
	}

	// 读取 status (UID, VmRSS, Threads)
	c.readProcStatus(procPath+"/status", proc)

	// 计算内存百分比
	if c.totalMemory > 0 && proc.MemRSS > 0 {
		proc.MemPercent = float64(proc.MemRSS) / float64(c.totalMemory) * 100
	}

	// 获取用户名
	proc.User = c.getUsername(proc.PID)

	return proc
}

// readProcStatus 读取 /proc/[pid]/status
func (c *processCollector) readProcStatus(path string, proc *model_v2.ProcessMetrics) {
	lines, err := utils.ReadFileLines(path)
	if err != nil {
		return
	}

	for _, line := range lines {
		key, value := utils.ParseKeyValue(line)
		switch key {
		case "VmRSS":
			// VmRSS: 1234 kB
			proc.MemRSS = utils.ParseMemValue(value)
		case "Threads":
			proc.Threads, _ = strconv.Atoi(value)
		}
	}
}

// getUsername 获取用户名（带缓存）
func (c *processCollector) getUsername(pid int) string {
	// 读取 UID
	path := c.cfg.Paths.ProcRoot + "/" + strconv.Itoa(pid) + "/status"
	lines, err := utils.ReadFileLines(path)
	if err != nil {
		return ""
	}

	var uid int
	for _, line := range lines {
		if strings.HasPrefix(line, "Uid:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				uid, _ = strconv.Atoi(fields[1])
				break
			}
		}
	}

	// 从缓存获取
	c.userCacheMu.RLock()
	username, ok := c.userCache[uid]
	c.userCacheMu.RUnlock()

	if ok {
		return username
	}

	// 查询用户名
	u, err := user.LookupId(strconv.Itoa(uid))
	if err != nil {
		username = strconv.Itoa(uid)
	} else {
		username = u.Username
	}

	// 更新缓存
	c.userCacheMu.Lock()
	c.userCache[uid] = username
	c.userCacheMu.Unlock()

	return username
}

type parseError struct {
	msg string
}

func (e *parseError) Error() string {
	return e.msg
}

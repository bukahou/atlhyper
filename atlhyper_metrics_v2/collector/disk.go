package collector

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"AtlHyper/atlhyper_metrics_v2/config"
	"AtlHyper/atlhyper_metrics_v2/utils"
	"AtlHyper/model_v2"
)

// 虚拟文件系统类型（需要过滤）
var virtualFSTypes = map[string]bool{
	"sysfs": true, "proc": true, "devtmpfs": true, "devpts": true,
	"tmpfs": true, "securityfs": true, "cgroup": true, "cgroup2": true,
	"pstore": true, "debugfs": true, "hugetlbfs": true, "mqueue": true,
	"fusectl": true, "configfs": true, "binfmt_misc": true,
	"autofs": true, "overlay": true, "squashfs": true,
}

// diskCollector 磁盘采集器实现
type diskCollector struct {
	cfg *config.Config

	mu      sync.RWMutex
	metrics []model_v2.DiskMetrics

	// 采样数据
	prevSample *utils.DiskRawSample
	currSample *utils.DiskRawSample

	// 控制
	stopCh chan struct{}
	wg     sync.WaitGroup
}

// NewDiskCollector 创建磁盘采集器
func NewDiskCollector(cfg *config.Config) DiskCollector {
	return &diskCollector{
		cfg:    cfg,
		stopCh: make(chan struct{}),
	}
}

// Start 启动后台采样
func (c *diskCollector) Start() {
	c.sample()

	c.wg.Add(1)
	go c.sampleLoop()
}

// Stop 停止后台采样
func (c *diskCollector) Stop() {
	close(c.stopCh)
	c.wg.Wait()
}

// sampleLoop 采样循环
func (c *diskCollector) sampleLoop() {
	defer c.wg.Done()

	ticker := time.NewTicker(c.cfg.Collect.CPUInterval) // 使用相同间隔
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
func (c *diskCollector) sample() {
	sample, err := c.readDiskStats()
	if err != nil {
		return
	}

	c.mu.Lock()
	c.prevSample = c.currSample
	c.currSample = sample
	c.mu.Unlock()
}

// Collect 采集磁盘指标
func (c *diskCollector) Collect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 读取挂载点
	mounts, err := c.readMounts()
	if err != nil {
		return err
	}

	c.metrics = make([]model_v2.DiskMetrics, 0, len(mounts))

	for _, mount := range mounts {
		disk := model_v2.DiskMetrics{
			Device:     mount.device,
			MountPoint: mount.mountPoint,
			FSType:     mount.fsType,
		}

		// 获取空间使用（使用实际路径）
		c.getSpaceUsage(&disk, mount.actualPath)

		// 计算 I/O 速率
		if c.prevSample != nil && c.currSample != nil {
			c.calculateIORate(&disk)
		}

		c.metrics = append(c.metrics, disk)
	}

	return nil
}

// Get 获取磁盘指标
func (c *diskCollector) Get() []model_v2.DiskMetrics {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.metrics
}

type mountInfo struct {
	device     string
	mountPoint string // 显示路径（宿主机视角）
	fsType     string
	actualPath string // 实际路径（用于 statfs）
}

// readMounts 读取挂载点
func (c *diskCollector) readMounts() ([]mountInfo, error) {
	path := c.cfg.Paths.ProcRoot + "/mounts"
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var mounts []mountInfo
	seen := make(map[string]bool)
	hostRoot := c.cfg.Paths.HostRoot

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 3 {
			continue
		}

		device := fields[0]
		mountPoint := fields[1]
		fsType := fields[2]

		// 过滤虚拟文件系统
		if virtualFSTypes[fsType] {
			continue
		}

		// 过滤非块设备
		if !strings.HasPrefix(device, "/dev/") {
			continue
		}

		// 过滤 kubelet 临时挂载（volume-subpaths、termination-log 等）
		if strings.Contains(mountPoint, "/kubelet/pods/") ||
			strings.Contains(mountPoint, "/termination-log") ||
			strings.Contains(mountPoint, "/etc/hosts") ||
			strings.Contains(mountPoint, "/etc/hostname") ||
			strings.Contains(mountPoint, "/etc/resolv.conf") {
			continue
		}

		// 计算实际路径和显示路径
		actualPath := mountPoint
		displayPath := mountPoint

		// 在容器中，挂载点可能已经是 /host_root/... 形式
		if hostRoot != "/" && strings.HasPrefix(mountPoint, hostRoot) {
			// 挂载点已经包含 hostRoot，直接使用
			actualPath = mountPoint
			// 显示时去掉 hostRoot 前缀，还原为宿主机视角
			displayPath = strings.TrimPrefix(mountPoint, hostRoot)
			if displayPath == "" {
				displayPath = "/"
			}
		} else if hostRoot != "/" {
			// 挂载点是宿主机视角，需要拼接 hostRoot
			actualPath = hostRoot + mountPoint
		}

		// 去重（基于显示路径）
		if seen[displayPath] {
			continue
		}
		seen[displayPath] = true

		mounts = append(mounts, mountInfo{
			device:     device,
			mountPoint: displayPath,  // 显示路径
			fsType:     fsType,
			actualPath: actualPath,   // 实际路径（用于 statfs）
		})
	}

	return mounts, scanner.Err()
}

// getSpaceUsage 获取空间使用
func (c *diskCollector) getSpaceUsage(disk *model_v2.DiskMetrics, actualPath string) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(actualPath, &stat); err != nil {
		return
	}

	disk.Total = int64(stat.Blocks) * int64(stat.Bsize)
	disk.Available = int64(stat.Bavail) * int64(stat.Bsize)
	disk.Used = disk.Total - int64(stat.Bfree)*int64(stat.Bsize)

	if disk.Total > 0 {
		disk.UsagePercent = float64(disk.Used) / float64(disk.Total) * 100
	}
}

// readDiskStats 读取 /proc/diskstats
func (c *diskCollector) readDiskStats() (*utils.DiskRawSample, error) {
	path := c.cfg.Paths.ProcRoot + "/diskstats"
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	sample := &utils.DiskRawSample{
		Timestamp: time.Now(),
		Stats:     make(map[string]utils.DiskRawStats),
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 14 {
			continue
		}

		device := fields[2]
		// 只统计主设备（排除分区）
		// 或者统计所有，让上层根据挂载点过滤

		sample.Stats[device] = utils.DiskRawStats{
			Device:        device,
			ReadComplete:  parseUint64(fields[3]),
			ReadSectors:   parseUint64(fields[5]),
			WriteComplete: parseUint64(fields[7]),
			WriteSectors:  parseUint64(fields[9]),
			IOInProgress:  parseUint64(fields[11]),
			IOTime:        parseUint64(fields[12]),
		}
	}

	return sample, scanner.Err()
}

// calculateIORate 计算 I/O 速率
func (c *diskCollector) calculateIORate(disk *model_v2.DiskMetrics) {
	// 从设备路径提取设备名
	deviceName := strings.TrimPrefix(disk.Device, "/dev/")

	prev, prevOK := c.prevSample.Stats[deviceName]
	curr, currOK := c.currSample.Stats[deviceName]

	if !prevOK || !currOK {
		return
	}

	elapsed := c.currSample.Timestamp.Sub(c.prevSample.Timestamp).Seconds()
	if elapsed <= 0 {
		return
	}

	// 扇区大小通常为 512 字节
	const sectorSize = 512

	// 读写字节
	disk.ReadBytes = int64(curr.ReadSectors) * sectorSize
	disk.WriteBytes = int64(curr.WriteSectors) * sectorSize

	// 读写速率 (bytes/s)
	readSectorsDelta := float64(curr.ReadSectors - prev.ReadSectors)
	writeSectorsDelta := float64(curr.WriteSectors - prev.WriteSectors)
	disk.ReadRate = readSectorsDelta * sectorSize / elapsed
	disk.WriteRate = writeSectorsDelta * sectorSize / elapsed

	// IOPS
	readIOPSDelta := float64(curr.ReadComplete - prev.ReadComplete)
	writeIOPSDelta := float64(curr.WriteComplete - prev.WriteComplete)
	disk.ReadIOPS = readIOPSDelta / elapsed
	disk.WriteIOPS = writeIOPSDelta / elapsed

	// I/O 利用率 (%)
	ioTimeDelta := float64(curr.IOTime - prev.IOTime)
	disk.IOUtil = utils.Clamp(ioTimeDelta/(elapsed*1000)*100, 0, 100)
}

func parseUint64(s string) uint64 {
	v, _ := strconv.ParseUint(s, 10, 64)
	return v
}

package collector

import (
	"bufio"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"AtlHyper/atlhyper_metrics_v2/config"
	"AtlHyper/atlhyper_metrics_v2/utils"
	"AtlHyper/model_v2"
)

// 虚拟网络接口前缀（需要过滤）
var virtualNetPrefixes = []string{
	"lo", "docker", "veth", "br-", "virbr", "vnet", "cni", "flannel", "cali",
}

// networkCollector 网络采集器实现
type networkCollector struct {
	cfg *config.Config

	mu      sync.RWMutex
	metrics []model_v2.NetworkMetrics

	// 采样数据
	prevSample *utils.NetRawSample
	currSample *utils.NetRawSample

	// 控制
	stopCh chan struct{}
	wg     sync.WaitGroup
}

// NewNetworkCollector 创建网络采集器
func NewNetworkCollector(cfg *config.Config) NetworkCollector {
	return &networkCollector{
		cfg:    cfg,
		stopCh: make(chan struct{}),
	}
}

// Start 启动后台采样
func (c *networkCollector) Start() {
	c.sample()

	c.wg.Add(1)
	go c.sampleLoop()
}

// Stop 停止后台采样
func (c *networkCollector) Stop() {
	close(c.stopCh)
	c.wg.Wait()
}

// sampleLoop 采样循环
func (c *networkCollector) sampleLoop() {
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
func (c *networkCollector) sample() {
	sample, err := c.readNetStats()
	if err != nil {
		return
	}

	c.mu.Lock()
	c.prevSample = c.currSample
	c.currSample = sample
	c.mu.Unlock()
}

// Collect 采集网络指标
func (c *networkCollector) Collect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.currSample == nil {
		return nil
	}

	// 获取网络接口信息
	interfaces, err := net.Interfaces()
	if err != nil {
		return err
	}

	c.metrics = make([]model_v2.NetworkMetrics, 0)

	for _, iface := range interfaces {
		// 过滤虚拟接口
		if c.isVirtualInterface(iface.Name) {
			continue
		}

		// 过滤 down 状态的接口
		if iface.Flags&net.FlagUp == 0 {
			continue
		}

		netMetric := model_v2.NetworkMetrics{
			Interface:  iface.Name,
			MACAddress: iface.HardwareAddr.String(),
			MTU:        iface.MTU,
		}

		// 获取 IP 地址
		addrs, _ := iface.Addrs()
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && ipnet.IP.To4() != nil {
				netMetric.IPAddress = ipnet.IP.String()
				break
			}
		}

		// 获取状态和速度
		c.getInterfaceStatus(&netMetric)

		// 获取流量统计
		if stats, ok := c.currSample.Stats[iface.Name]; ok {
			netMetric.RxBytes = int64(stats.RxBytes)
			netMetric.TxBytes = int64(stats.TxBytes)
			netMetric.RxPackets = int64(stats.RxPackets)
			netMetric.TxPackets = int64(stats.TxPackets)
			netMetric.RxErrors = int64(stats.RxErrors)
			netMetric.TxErrors = int64(stats.TxErrors)
			netMetric.RxDropped = int64(stats.RxDropped)
			netMetric.TxDropped = int64(stats.TxDropped)
		}

		// 计算速率
		if c.prevSample != nil {
			c.calculateNetRate(&netMetric)
		}

		c.metrics = append(c.metrics, netMetric)
	}

	return nil
}

// Get 获取网络指标
func (c *networkCollector) Get() []model_v2.NetworkMetrics {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.metrics
}

// isVirtualInterface 检查是否为虚拟接口
func (c *networkCollector) isVirtualInterface(name string) bool {
	for _, prefix := range virtualNetPrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

// getInterfaceStatus 获取接口状态和速度
func (c *networkCollector) getInterfaceStatus(netMetric *model_v2.NetworkMetrics) {
	sysPath := c.cfg.Paths.SysRoot + "/class/net/" + netMetric.Interface

	// 状态
	if state, err := utils.ReadFileString(sysPath + "/operstate"); err == nil {
		netMetric.Status = state
	}

	// 速度 (Mbps)
	if speed, err := utils.ReadFileUint64(sysPath + "/speed"); err == nil {
		netMetric.Speed = int64(speed)
	}
}

// readNetStats 读取 /proc/net/dev
func (c *networkCollector) readNetStats() (*utils.NetRawSample, error) {
	path := c.cfg.Paths.ProcRoot + "/net/dev"
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	sample := &utils.NetRawSample{
		Timestamp: time.Now(),
		Stats:     make(map[string]utils.NetRawStats),
	}

	scanner := bufio.NewScanner(file)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		if lineNum <= 2 {
			continue // 跳过头两行
		}

		line := scanner.Text()
		// 格式: iface: rx_bytes rx_packets rx_errs rx_drop ... tx_bytes tx_packets tx_errs tx_drop ...
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		iface := strings.TrimSpace(parts[0])
		fields := strings.Fields(parts[1])
		if len(fields) < 16 {
			continue
		}

		sample.Stats[iface] = utils.NetRawStats{
			Interface: iface,
			RxBytes:   parseUint64(fields[0]),
			RxPackets: parseUint64(fields[1]),
			RxErrors:  parseUint64(fields[2]),
			RxDropped: parseUint64(fields[3]),
			TxBytes:   parseUint64(fields[8]),
			TxPackets: parseUint64(fields[9]),
			TxErrors:  parseUint64(fields[10]),
			TxDropped: parseUint64(fields[11]),
		}
	}

	return sample, scanner.Err()
}

// calculateNetRate 计算网络速率
func (c *networkCollector) calculateNetRate(netMetric *model_v2.NetworkMetrics) {
	prev, prevOK := c.prevSample.Stats[netMetric.Interface]
	curr, currOK := c.currSample.Stats[netMetric.Interface]

	if !prevOK || !currOK {
		return
	}

	elapsed := c.currSample.Timestamp.Sub(c.prevSample.Timestamp).Seconds()
	if elapsed <= 0 {
		return
	}

	netMetric.RxRate = float64(curr.RxBytes-prev.RxBytes) / elapsed
	netMetric.TxRate = float64(curr.TxBytes-prev.TxBytes) / elapsed
}

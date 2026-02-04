// Package atlhyper_metrics_v2 节点指标采集器核心包
//
// 本包提供 Metrics 的启动器，负责:
//   - 初始化所有采集器 (CPU, Memory, Disk, Network, Temperature, Process)
//   - 聚合采集数据为 NodeMetricsSnapshot
//   - 定时推送数据到 Agent
//   - 生命周期管理 (启动、运行、停止)
//
// 使用方式:
//
//	config.Load()
//	metrics := atlhyper_metrics_v2.New()
//	metrics.Run(ctx)
package atlhyper_metrics_v2

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"AtlHyper/atlhyper_metrics_v2/aggregator"
	"AtlHyper/atlhyper_metrics_v2/config"
	"AtlHyper/atlhyper_metrics_v2/pusher"
	"AtlHyper/common/logger"
)

var log = logger.Module("Metrics")

// Metrics 是节点指标采集器的主结构体
type Metrics struct {
	cfg        *config.Config
	aggregator *aggregator.Aggregator
	pusher     *pusher.HTTPPusher
}

// New 创建并初始化 Metrics 实例
func New() *Metrics {
	cfg := &config.GlobalConfig

	log.Info("节点指标采集器初始化中",
		"node", cfg.NodeName,
		"agent", cfg.Push.AgentAddr,
		"interval", cfg.Collect.CollectInterval,
	)

	// 创建聚合器
	agg := aggregator.New(cfg)

	// 创建推送器
	push := pusher.New(cfg)

	return &Metrics{
		cfg:        cfg,
		aggregator: agg,
		pusher:     push,
	}
}

// Run 运行采集器
func (m *Metrics) Run(ctx context.Context) error {
	// 启动后台采样
	m.aggregator.Start()
	log.Info("后台采样已启动")

	// 等待采样数据就绪
	time.Sleep(2 * time.Second)

	// 定时采集 + 推送
	ticker := time.NewTicker(m.cfg.Collect.CollectInterval)
	defer ticker.Stop()

	// 信号处理
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	log.Info("开始采集循环")

	for {
		select {
		case <-ctx.Done():
			log.Info("上下文取消，正在停止...")
			return m.Stop()

		case <-sigCh:
			log.Info("收到退出信号，正在停止...")
			return m.Stop()

		case <-ticker.C:
			m.collectAndPush()
		}
	}
}

// collectAndPush 执行一次采集和推送
func (m *Metrics) collectAndPush() {
	// 采集
	snapshot, err := m.aggregator.Collect()
	if err != nil {
		log.Error("采集失败", "err", err)
		return
	}

	// 推送
	if err := m.pusher.Push(snapshot); err != nil {
		log.Error("推送失败", "err", err)
		return
	}

	log.Debug("采集并推送成功",
		"cpu", snapshot.CPU.UsagePercent,
		"mem", snapshot.Memory.UsagePercent,
	)
}

// Stop 停止采集器
func (m *Metrics) Stop() error {
	m.aggregator.Stop()
	log.Info("采集器已停止")
	return nil
}

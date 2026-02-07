// atlhyper_master_v2/slo/aggregator.go
// SLO 小时聚合器（过渡版本）
//
// Master P3 将完全重写以支持三层数据（service + edge + ingress）
// 和 JSON bucket 格式的聚合计算。
package slo

import (
	"log"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
)

// Aggregator 小时聚合器
type Aggregator struct {
	repo     database.SLORepository
	interval time.Duration
	stopCh   chan struct{}
	doneCh   chan struct{}
}

// NewAggregator 创建聚合器
func NewAggregator(repo database.SLORepository, interval time.Duration) *Aggregator {
	return &Aggregator{
		repo:     repo,
		interval: interval,
		stopCh:   make(chan struct{}),
		doneCh:   make(chan struct{}),
	}
}

// Start 启动聚合器
func (a *Aggregator) Start() {
	go a.run()
	log.Printf("[SLO Aggregator] 已启动（过渡版本），聚合间隔: %v", a.interval)
}

// Stop 停止聚合器
func (a *Aggregator) Stop() {
	close(a.stopCh)
	<-a.doneCh
	log.Println("[SLO Aggregator] 已停止")
}

func (a *Aggregator) run() {
	defer close(a.doneCh)

	ticker := time.NewTicker(a.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// TODO(Master P3): 实现三层数据聚合
			// - service raw → service hourly
			// - edge raw → edge hourly
			// - ingress raw → ingress hourly
		case <-a.stopCh:
			return
		}
	}
}

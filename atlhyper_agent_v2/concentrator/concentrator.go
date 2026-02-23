// Package concentrator 本地时序聚合器（类 Datadog Concentrator）
//
// 维护最近 1 小时的降采样数据，每次快照时输出预聚合时序。
// 采用 1 分钟粒度环形缓冲，Agent 每 10s 采集一次，
// 同一分钟内多次写入取最新值（GAUGE 语义）。
package concentrator

import (
	"sync"
	"time"

	"AtlHyper/model_v3/cluster"
	"AtlHyper/model_v3/metrics"
	"AtlHyper/model_v3/slo"
)

const ringCapacity = 60 // 60 分钟 = 1 小时

// Concentrator 本地时序聚合器
type Concentrator struct {
	nodeRings map[string]*nodeMetricsRing
	sloRings  map[string]*sloMetricsRing
	mu        sync.RWMutex
}

// New 创建 Concentrator
func New() *Concentrator {
	return &Concentrator{
		nodeRings: make(map[string]*nodeMetricsRing),
		sloRings:  make(map[string]*sloMetricsRing),
	}
}

// Ingest 摄入当前 OTel 快照数据，更新时序环
func (c *Concentrator) Ingest(nodes []metrics.NodeMetrics, sloIngress []slo.IngressSLO, ts time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()

	minute := alignMinute(ts)

	// 节点指标
	for i := range nodes {
		node := &nodes[i]
		ring, ok := c.nodeRings[node.NodeName]
		if !ok {
			ring = &nodeMetricsRing{}
			c.nodeRings[node.NodeName] = ring
		}

		var diskPct float64
		if d := node.GetPrimaryDisk(); d != nil {
			diskPct = d.UsagePct
		}
		var netRx, netTx float64
		for _, n := range node.Networks {
			if n.Up && n.Interface != "lo" {
				netRx += n.RxBytesPerSec
				netTx += n.TxBytesPerSec
			}
		}

		ring.put(minute, cluster.NodeMetricsPoint{
			Timestamp: ts.Truncate(time.Minute),
			CPUPct:    node.CPU.UsagePct,
			MemPct:    node.Memory.UsagePct,
			DiskPct:   diskPct,
			NetRxBps:  netRx,
			NetTxBps:  netTx,
			Load1:     node.CPU.Load1,
		})
	}

	// SLO Ingress
	for i := range sloIngress {
		svc := &sloIngress[i]
		ring, ok := c.sloRings[svc.ServiceKey]
		if !ok {
			ring = &sloMetricsRing{}
			c.sloRings[svc.ServiceKey] = ring
		}
		ring.put(minute, cluster.SLOTimePoint{
			Timestamp:   ts.Truncate(time.Minute),
			RPS:         svc.RPS,
			SuccessRate: svc.SuccessRate,
			P99Ms:       svc.P99Ms,
		})
	}
}

// FlushNodeSeries 输出所有节点的预聚合时序
func (c *Concentrator) FlushNodeSeries() []cluster.NodeMetricsTimeSeries {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make([]cluster.NodeMetricsTimeSeries, 0, len(c.nodeRings))
	now := alignMinute(time.Now())
	for name, ring := range c.nodeRings {
		points := ring.flush(now)
		if len(points) > 0 {
			result = append(result, cluster.NodeMetricsTimeSeries{
				NodeName: name,
				Points:   points,
			})
		}
	}
	return result
}

// FlushSLOSeries 输出所有服务的预聚合 SLO 时序
func (c *Concentrator) FlushSLOSeries() []cluster.SLOServiceTimeSeries {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make([]cluster.SLOServiceTimeSeries, 0, len(c.sloRings))
	now := alignMinute(time.Now())
	for name, ring := range c.sloRings {
		points := ring.flush(now)
		if len(points) > 0 {
			result = append(result, cluster.SLOServiceTimeSeries{
				ServiceName: name,
				Points:      points,
			})
		}
	}
	return result
}

// alignMinute 截断到分钟边界（Unix 分钟数）
func alignMinute(ts time.Time) int64 {
	return ts.Unix() / 60
}

// ================================================================
// nodeMetricsRing — 节点指标环形缓冲
// ================================================================

type nodeMetricsRing struct {
	points  [ringCapacity]cluster.NodeMetricsPoint
	minutes [ringCapacity]int64 // 对应分钟时间戳
}

func (r *nodeMetricsRing) put(minute int64, pt cluster.NodeMetricsPoint) {
	idx := int(minute % ringCapacity)
	r.minutes[idx] = minute
	r.points[idx] = pt
}

func (r *nodeMetricsRing) flush(nowMinute int64) []cluster.NodeMetricsPoint {
	result := make([]cluster.NodeMetricsPoint, 0, ringCapacity)
	for i := 0; i < ringCapacity; i++ {
		// 从最旧到最新
		m := nowMinute - int64(ringCapacity-1) + int64(i)
		idx := int(m % ringCapacity)
		if idx < 0 {
			idx += ringCapacity
		}
		if r.minutes[idx] == m {
			result = append(result, r.points[idx])
		}
	}
	return result
}

// ================================================================
// sloMetricsRing — SLO 指标环形缓冲
// ================================================================

type sloMetricsRing struct {
	points  [ringCapacity]cluster.SLOTimePoint
	minutes [ringCapacity]int64
}

func (r *sloMetricsRing) put(minute int64, pt cluster.SLOTimePoint) {
	idx := int(minute % ringCapacity)
	r.minutes[idx] = minute
	r.points[idx] = pt
}

func (r *sloMetricsRing) flush(nowMinute int64) []cluster.SLOTimePoint {
	result := make([]cluster.SLOTimePoint, 0, ringCapacity)
	for i := 0; i < ringCapacity; i++ {
		m := nowMinute - int64(ringCapacity-1) + int64(i)
		idx := int(m % ringCapacity)
		if idx < 0 {
			idx += ringCapacity
		}
		if r.minutes[idx] == m {
			result = append(result, r.points[idx])
		}
	}
	return result
}

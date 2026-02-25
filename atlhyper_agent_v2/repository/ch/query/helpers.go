// Package query ClickHouse 按需查询仓库实现
package query

import (
	"math"
	"time"

	"AtlHyper/model_v3/metrics"
)

// rawPoint ClickHouse 原始时序数据点（用于 rate 计算）
type rawPoint struct {
	Time  time.Time
	Value float64
}

// computeRateSeries 从累积 counter 序列计算逐点速率
//
// 对于 ClickHouse otel_metrics_sum 表中的 counter 类型指标（如 CPU seconds、网络字节数），
// 需要将累积值转换为每秒速率。
//
// 处理 counter reset: 如果后一个值小于前一个值，视为计数器重置，使用后一个值作为增量。
func computeRateSeries(points []rawPoint) []metrics.Point {
	if len(points) < 2 {
		return nil
	}
	result := make([]metrics.Point, 0, len(points)-1)
	for i := 1; i < len(points); i++ {
		dt := points[i].Time.Sub(points[i-1].Time).Seconds()
		if dt <= 0 {
			continue
		}
		delta := points[i].Value - points[i-1].Value
		if delta < 0 {
			delta = points[i].Value // counter reset
		}
		result = append(result, metrics.Point{
			Timestamp: points[i].Time,
			Value:     delta / dt,
		})
	}
	return result
}

// computeRate 从两个端点计算速率（用于快照查询）
//
// 适用于 argMax/argMin 模式：
//
//	rate = (max_value - min_value) / (max_time - min_time)
func computeRate(minVal, maxVal float64, minTime, maxTime time.Time) float64 {
	dt := maxTime.Sub(minTime).Seconds()
	if dt <= 0 {
		return 0
	}
	delta := maxVal - minVal
	if delta < 0 {
		delta = maxVal // counter reset
	}
	return delta / dt
}

// histogramPercentile 从 Prometheus-style histogram 计算百分位数
//
// bounds: 桶边界 (ExplicitBounds)，升序排列
// counts: 每个桶的计数 (BucketCounts)，len(counts) == len(bounds) + 1
// p: 百分位数 (0.0 - 1.0)
//
// 使用线性插值估算百分位值。最后一个桶（+Inf）使用最后一个边界值。
func histogramPercentile(bounds []float64, counts []uint64, p float64) float64 {
	if len(bounds) == 0 || len(counts) == 0 {
		return 0
	}
	if p <= 0 {
		return 0
	}
	if p >= 1 {
		if len(bounds) > 0 {
			return bounds[len(bounds)-1]
		}
		return 0
	}

	// 计算总数
	var total uint64
	for _, c := range counts {
		total += c
	}
	if total == 0 {
		return 0
	}

	target := float64(total) * p

	// 累积计数找到目标桶
	var cumulative uint64
	for i, c := range counts {
		cumulative += c
		if float64(cumulative) >= target {
			// 在第 i 个桶中找到了
			var lower, upper float64
			var prevCumulative uint64

			if i == 0 {
				lower = 0
			} else {
				lower = bounds[i-1]
				prevCumulative = cumulative - c
			}

			if i < len(bounds) {
				upper = bounds[i]
			} else {
				// +Inf 桶，使用最后一个边界
				if len(bounds) > 0 {
					return bounds[len(bounds)-1]
				}
				return 0
			}

			// 线性插值
			if c == 0 {
				return lower
			}
			fraction := (target - float64(prevCumulative)) / float64(c)
			return lower + fraction*(upper-lower)
		}
	}

	// 不应到达这里
	if len(bounds) > 0 {
		return bounds[len(bounds)-1]
	}
	return 0
}

// parseDurationNanos 将纳秒 Duration 转换为毫秒
func parseDurationNanos(nanos int64) float64 {
	return float64(nanos) / 1e6
}

// safeDiv 安全除法，避免除零
func safeDiv(a, b float64) float64 {
	if b == 0 {
		return 0
	}
	return a / b
}

// safeDivPct 安全百分比计算
func safeDivPct(a, b float64) float64 {
	if b == 0 {
		return 0
	}
	return (a / b) * 100
}

// clamp 将值限制在 [min, max] 范围
func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// roundTo 四舍五入到指定小数位（NaN/Inf → 0）
func roundTo(v float64, decimals int) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	pow := math.Pow(10, float64(decimals))
	return math.Round(v*pow) / pow
}

// roundRPS 对 RPS 值智能舍入
//
// 大窗口（如 1d/7d/30d）下 RPS 可能非常小（如 373/86400 = 0.004），
// 固定 2 位小数会丢失为 0。此函数保留至少 2 位有效数字：
//
//	1.234  → 1.23  (2 位小数足够)
//	0.123  → 0.12
//	0.0043 → 0.0043 (自动扩展到 4 位)
func roundRPS(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) || v <= 0 {
		return 0
	}
	// 至少保留 2 位有效数字
	digits := 2
	if v < 1 {
		digits = 2 - int(math.Floor(math.Log10(v)))
		if digits > 6 {
			digits = 6
		}
	}
	pow := math.Pow(10, float64(digits))
	return math.Round(v*pow) / pow
}

// cumulativeToDifferential 将 Prometheus 风格累积桶计数转换为差分计数
//
// 输入: [10, 25, 40, 50] (cumulative: ≤b0, ≤b1, ≤b2, ≤b3)
// 输出: [10, 15, 15, 10] (differential: [0,b0], (b0,b1], (b1,b2], (b2,b3])
func cumulativeToDifferential(counts []uint64) []uint64 {
	if len(counts) == 0 {
		return counts
	}
	diff := make([]uint64, len(counts))
	diff[0] = counts[0]
	for i := 1; i < len(counts); i++ {
		if counts[i] >= counts[i-1] {
			diff[i] = counts[i] - counts[i-1]
		}
		// else: counter anomaly, keep 0
	}
	return diff
}

// sinceInterval 将 time.Duration 转换为 ClickHouse INTERVAL 秒数
func sinceSeconds(since time.Duration) int64 {
	s := int64(since.Seconds())
	if s <= 0 {
		s = 300 // 默认 5 分钟
	}
	return s
}

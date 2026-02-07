// atlhyper_master_v2/slo/calculator.go
// SLO 核心算法模块
package slo

import (
	"math"
	"sort"
)

// CalculateDelta 计算 Counter 增量（处理 reset）
// 当 Pod 重启时，Counter 会重置为 0，需要检测并正确计算增量
func CalculateDelta(newValue, prevValue int64) int64 {
	if newValue >= prevValue {
		// 正常情况：Counter 递增
		return newValue - prevValue
	}
	// Counter 重置：新值小于旧值，增量 = 新值
	return newValue
}

// CalculateQuantile 计算 Histogram 分位数（线性插值）
// buckets: map[le]count，le 为 bucket 上界（秒）
// quantile: 目标分位数（如 0.95 表示 P95）
// 返回值单位：秒
func CalculateQuantile(buckets map[float64]int64, quantile float64) float64 {
	if len(buckets) == 0 {
		return 0
	}

	// 按 le 排序
	les := make([]float64, 0, len(buckets))
	for le := range buckets {
		les = append(les, le)
	}
	sort.Float64s(les)

	// 获取总数（+Inf bucket 包含所有请求）
	// Prometheus histogram 的 bucket 是累积的，所以最大 bucket 的值就是总请求数
	total := buckets[math.Inf(1)]
	if total == 0 {
		// 尝试使用最大的非零 bucket 值作为 total
		// 从大到小遍历 le，找到第一个非零值
		for i := len(les) - 1; i >= 0; i-- {
			if !math.IsInf(les[i], 1) && buckets[les[i]] > 0 {
				total = buckets[les[i]]
				break
			}
		}
		if total == 0 {
			return 0
		}
	}

	// 计算目标位置
	rank := float64(total) * quantile

	// 找到目标区间并插值
	var prevLE float64 = 0
	var prevCount int64 = 0

	for _, le := range les {
		if math.IsInf(le, 1) {
			continue // 跳过 +Inf
		}
		count := buckets[le]
		if float64(count) >= rank {
			// 目标在 [prevLE, le] 区间内
			if count == prevCount {
				return le
			}
			// 线性插值
			fraction := (rank - float64(prevCount)) / float64(count-prevCount)
			return prevLE + (le-prevLE)*fraction
		}
		prevLE = le
		prevCount = count
	}

	// 返回最大非 +Inf 的 le
	for i := len(les) - 1; i >= 0; i-- {
		if !math.IsInf(les[i], 1) {
			return les[i]
		}
	}
	return 0
}

// CalculateQuantileMs 计算分位数并返回毫秒值
func CalculateQuantileMs(buckets map[float64]int64, quantile float64) int {
	seconds := CalculateQuantile(buckets, quantile)
	return int(seconds * 1000)
}

// CalculateAvailability 计算可用性
// 返回值：百分比（如 99.95 表示 99.95%）
func CalculateAvailability(totalRequests, errorRequests int64) float64 {
	if totalRequests == 0 {
		return 100.0 // 无请求时视为 100% 可用
	}
	successRequests := totalRequests - errorRequests
	if successRequests < 0 {
		successRequests = 0
	}
	return float64(successRequests) / float64(totalRequests) * 100
}

// CalculateErrorRate 计算错误率
// 返回值：百分比（如 0.05 表示 0.05%）
func CalculateErrorRate(totalRequests, errorRequests int64) float64 {
	if totalRequests == 0 {
		return 0
	}
	return float64(errorRequests) / float64(totalRequests) * 100
}

// CalculateErrorBudgetRemaining 计算错误预算剩余
// actualAvail: 实际可用性（百分比）
// targetAvail: 目标可用性（百分比）
// 返回值：剩余预算百分比（100 表示 100% 剩余，0 表示已用完，负数表示超支）
func CalculateErrorBudgetRemaining(actualAvail, targetAvail float64) float64 {
	errorBudget := 100 - targetAvail // 允许的错误率
	consumed := 100 - actualAvail    // 实际消耗的错误率

	if errorBudget <= 0 {
		// 目标是 100% 可用性
		if consumed <= 0 {
			return 100.0
		}
		return 0.0
	}

	remaining := ((errorBudget - consumed) / errorBudget) * 100

	// 限制在合理范围
	if remaining > 100 {
		return 100
	}
	if remaining < -100 {
		return -100
	}
	return remaining
}

// DetermineStatus 判断 SLO 状态
// 返回值：healthy / warning / critical
func DetermineStatus(actualAvail, targetAvail float64, actualP95, targetP95 int) string {
	// 可用性未达标 -> critical
	if actualAvail < targetAvail {
		return "critical"
	}
	// 延迟超标 -> warning
	if actualP95 > targetP95 {
		return "warning"
	}
	return "healthy"
}

// CalculateTrend 计算趋势
// 返回值：up / down / stable
func CalculateTrend(currentAvail, previousAvail float64) string {
	diff := currentAvail - previousAvail
	threshold := 0.1 // 0.1% 变化阈值

	if diff > threshold {
		return "up"
	}
	if diff < -threshold {
		return "down"
	}
	return "stable"
}

// CalculateRPS 计算每秒请求数
// totalRequests: 时间窗口内的总请求数
// durationSeconds: 时间窗口秒数
func CalculateRPS(totalRequests int64, durationSeconds float64) float64 {
	if durationSeconds <= 0 {
		return 0
	}
	return float64(totalRequests) / durationSeconds
}

// MergeBuckets 合并多个 histogram bucket
// 用于聚合多个采样点的 bucket 数据
func MergeBuckets(bucketsList ...map[float64]int64) map[float64]int64 {
	result := make(map[float64]int64)
	for _, buckets := range bucketsList {
		for le, count := range buckets {
			result[le] += count
		}
	}
	return result
}

// ParseJSONBuckets 解析 JSON bucket 字符串为 map[float64]int64
// JSON 格式: {"100":10, "500":50, ...}（key 为毫秒字符串）
// 返回 map key 为秒值（兼容 CalculateQuantile）
func ParseJSONBuckets(jsonStr string) map[float64]int64 {
	if jsonStr == "" {
		return nil
	}
	// TODO(Master P3): 实现 JSON 解析
	return nil
}

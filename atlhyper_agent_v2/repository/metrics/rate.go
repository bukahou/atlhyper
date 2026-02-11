package metrics

// counterRate 计算 counter 类型指标的速率
//
// 处理 counter reset（cur < prev 时返回 0）和 elapsed=0 的情况。
func counterRate(cur, prev, elapsed float64) float64 {
	if elapsed <= 0 || cur < prev {
		return 0
	}
	return (cur - prev) / elapsed
}

// counterDelta 计算 counter 类型指标的增量
//
// 处理 counter reset（cur < prev 时返回 0）。
func counterDelta(cur, prev float64) float64 {
	if cur < prev {
		return 0
	}
	return cur - prev
}

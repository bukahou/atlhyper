// external/uiapi/metrics/utils.go
package metrics

import (
	"strconv"
	"time"

	model "NeuroController/model/metrics"
)

// parseSince 解析 since 查询参数，支持 RFC3339/RFC3339Nano 和持续时间（如 "15m"、"1h"）
// 返回值：
//   - time.Time：解析结果（UTC）
//   - bool：true 表示成功，false 表示未传或格式错误
func parseSince(v string) (time.Time, bool) {
	if v == "" {
		return time.Time{}, false
	}
	if t, err := time.Parse(time.RFC3339Nano, v); err == nil {
		return t.UTC(), true
	}
	if t, err := time.Parse(time.RFC3339, v); err == nil {
		return t.UTC(), true
	}
	if d, err := time.ParseDuration(v); err == nil {
		return time.Now().UTC().Add(-d), true
	}
	return time.Time{}, false
}

// parseLimit 解析 limit 查询参数，>0 才生效
func parseLimit(v string) int {
	if v == "" {
		return 0
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return 0
	}
	return n
}

// applyFilters 对节点快照数组进行过滤（先按时间，再按数量）
func applyFilters(in []*model.NodeMetricsSnapshot, hasSince bool, since time.Time, limit int) []*model.NodeMetricsSnapshot {
	if len(in) == 0 {
		return in
	}
	out := in

	// 时间过滤
	if hasSince {
		idx := 0
		found := false
		for i := range in {
			t := snapTime(in[i])
			if !t.IsZero() && (t.After(since) || t.Equal(since)) {
				idx = i
				found = true
				break
			}
		}
		if !found {
			return nil
		}
		out = in[idx:]
	}

	// 数量限制
	if limit > 0 && len(out) > limit {
		out = out[len(out)-limit:]
	}
	return out
}

// snapTime 安全获取快照的时间
func snapTime(s *model.NodeMetricsSnapshot) time.Time {
	if s == nil {
		return time.Time{}
	}
	switch v := any(s.Timestamp).(type) {
	case time.Time:
		return v.UTC()
	case *time.Time:
		if v == nil {
			return time.Time{}
		}
		return v.UTC()
	case string:
		if t, err := time.Parse(time.RFC3339Nano, v); err == nil {
			return t.UTC()
		}
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			return t.UTC()
		}
		return time.Time{}
	default:
		return time.Time{}
	}
}

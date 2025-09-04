package overview

import (
	"strings"
	"time"

	event "AtlHyper/model/event"
	"AtlHyper/model/metrics"
)

// 资源趋势：按分钟取峰值并记录峰值节点
func buildResourceUsageTrends(snaps []metrics.NodeMetricsSnapshot, since, until time.Time) []ResourceTrendPoint {
    // 统一用 UTC 做桶边界
    start := ceilToBucket(since.UTC(), time.Minute) // ✅ 起点向上取整
    end   := until.UTC()                            // 区间上界仍为开区间

    points := make(map[time.Time]*ResourceTrendPoint)

    for _, s := range snaps {
        bucket := floorToBucket(s.Timestamp.UTC(), time.Minute)
        // 只要 [start, end) 区间内的桶
        if bucket.Before(start) || !bucket.Before(end) {
            continue
        }
        p, ok := points[bucket]
        if !ok {
            p = &ResourceTrendPoint{At: bucket}
            points[bucket] = p
        }
        // 峰值统计（CPU / Mem / 温度）……
        if s.CPU.Usage > p.CPUPeak {
            p.CPUPeak = s.CPU.Usage
            p.CPUPeakNode = s.NodeName
        }
        if s.Memory.Total > 0 {
            memRatio := float64(s.Memory.Used) / float64(s.Memory.Total)
            if memRatio > p.MemPeak {
                p.MemPeak = memRatio
                p.MemPeakNode = s.NodeName
            }
        }
        if s.Temperature.CPUDegrees > p.TempPeak {
            p.TempPeak = s.Temperature.CPUDegrees
            p.TempPeakNode = s.NodeName
        }
    }

    // 精确容量：恰好 (end - start)/1m 个
    n := int(end.Sub(start) / time.Minute)
    out := make([]ResourceTrendPoint, 0, n)

    // 补齐每分钟桶：t ∈ [start, end)
    for t := start; t.Before(end); t = t.Add(time.Minute) {
        if p, ok := points[t]; ok {
            out = append(out, *p)
        } else {
            out = append(out, ResourceTrendPoint{At: t})
        }
    }
    return out
}


// 辅助函数：向上取整到整分（或任意粒度）
func ceilToBucket(t time.Time, d time.Duration) time.Time {
    tt := t.Truncate(d)
    if tt.Before(t) {
        return tt.Add(d)
    }
    return tt
}


// Alerts 趋势（24h）：按小时分桶统计 critical/warning/info
func buildAlertHourlyFromEvents(events []event.LogEvent, since, until time.Time) []AlertHourlyPoint {
	// 先桶化
	type acc struct{ c, w, i int }
	bm := map[time.Time]*acc{}

	for _, e := range events {
		t := e.Timestamp.UTC()
		if t.Before(since) || !t.Before(until) {
			continue
		}
		key := floorToBucket(t, time.Hour)
		a := bm[key]
		if a == nil {
			a = &acc{}
			bm[key] = a
		}
		switch strings.ToLower(e.Severity) {
		case "critical":
			a.c++
		case "warning":
			a.w++
		default:
			a.i++
		}
	}

	// 再按时间顺序补齐 24 个桶
	out := make([]AlertHourlyPoint, 0, int(until.Sub(since)/time.Hour))
	for t := floorToBucket(since, time.Hour); t.Before(until); t = t.Add(time.Hour) {
		a := bm[t]
		if a == nil {
			out = append(out, AlertHourlyPoint{At: t})
		} else {
			out = append(out, AlertHourlyPoint{
				At:       t,
				Critical: a.c,
				Warning:  a.w,
				Info:     a.i,
			})
		}
	}
	return out
}

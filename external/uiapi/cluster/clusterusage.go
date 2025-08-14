package cluster

import (
	"NeuroController/external/metrics_store"
	model "NeuroController/model/metrics"
	"sort"
	"time"
)

func computeClusterUsageSeriesAll(
	now time.Time,
	step time.Duration,
) (
	cpuSeries [][]float64,
	memSeries [][]float64,
	tempSeries [][]float64, // 每 3 分钟：该桶内全集群 CPU 最高温（℃）
) {
	const (
		maxWindow = 30 * time.Minute
		defStep   = 3 * time.Minute
		maxPoints = 10
	)

	// 默认值
	if now.IsZero() {
		now = time.Now()
	}
	if step <= 0 {
		step = defStep
	}

	snaps := metrics_store.SnapshotInMemoryMetrics()

	// 预处理：各节点快照按时间升序；记录最早的快照时间（用 UTC 比较更稳）
	var earliestUTC time.Time
	earliestSet := false
	for _, arr := range snaps {
		if len(arr) == 0 {
			continue
		}
		sort.Slice(arr, func(i, j int) bool { return arr[i].Timestamp.Before(arr[j].Timestamp) })
		firstUTC := arr[0].Timestamp.UTC()
		if !earliestSet || firstUTC.Before(earliestUTC) {
			earliestUTC = firstUTC
			earliestSet = true
		}
	}
	if !earliestSet {
		return
	}

	// —— 本地时区对齐（例如在日本运行则为 Asia/Tokyo；如需强制东京，用 LoadLocation("Asia/Tokyo")）——
	loc := time.Local                // 或：loc, _ := time.LoadLocation("Asia/Tokyo")
	localNow := now.In(loc)
	// 末端对齐到“下一个 step 边界”（例如 16:41、step=3m => 16:42）
	localEndAligned := localNow.Truncate(step).Add(step)

	// 仅取最近 maxPoints 个桶
	window := time.Duration(maxPoints) * step
	if window > maxWindow {
		window = maxWindow
	}
	localFrom := localEndAligned.Add(-window)

	// 用 UTC 与快照比较；起点不早于最早快照
	endAlignedUTC := localEndAligned.UTC()
	fromUTC := localFrom.UTC()
	if fromUTC.Before(earliestUTC) {
		fromUTC = earliestUTC
	}

	// helper：找 <= t 的最近一条（LOCF）
	getLatestBefore := func(arr []*model.NodeMetricsSnapshot, t time.Time) *model.NodeMetricsSnapshot {
		if len(arr) == 0 {
			return nil
		}
		i := sort.Search(len(arr), func(i int) bool {
			return !arr[i].Timestamp.Before(t) // 第一个 >= t
		})
		if i == 0 {
			return nil
		}
		return arr[i-1]
	}

	// 逐桶聚合：最多 10 组
	points := 0
	for bucketEndUTC := fromUTC.Add(step); !bucketEndUTC.After(endAlignedUTC) && points < maxPoints; bucketEndUTC = bucketEndUTC.Add(step) {
		var cpuNum, cpuDen float64
		var memUsed, memTot uint64
		var maxCPUTemp float64
		var covered, tempCovered int

		for _, arr := range snaps {
			s := getLatestBefore(arr, bucketEndUTC)
			if s == nil {
				continue
			}

			// CPU：核数加权
			cpuNum += s.CPU.Usage * float64(s.CPU.Cores)
			cpuDen += float64(s.CPU.Cores)

			// 内存：总使用 / 总容量
			memUsed += s.Memory.Used
			memTot += s.Memory.Total

			// 温度：该桶内全集群最大 CPU 温
			if tempCovered == 0 || s.Temperature.CPUDegrees > maxCPUTemp {
				maxCPUTemp = s.Temperature.CPUDegrees
			}
			tempCovered++
			covered++
		}

		if covered == 0 {
			// 如需补 0 点，可在此处改为追加 0 值。
			continue
		}

		var cpuPct, memPct float64
		if cpuDen > 0 {
			cpuPct = (cpuNum / cpuDen) * 100.0
		}
		if memTot > 0 {
			memPct = (float64(memUsed) / float64(memTot)) * 100.0
		}

		// ts 用“本地桶结束时刻”的 epoch ms，便于前端按本地时区展示
		localBucketEnd := bucketEndUTC.In(loc)
		tsMillis := float64(localBucketEnd.UnixMilli())

		cpuSeries = append(cpuSeries, []float64{tsMillis, cpuPct})
		memSeries = append(memSeries, []float64{tsMillis, memPct})
		if tempCovered > 0 {
			tempSeries = append(tempSeries, []float64{tsMillis, maxCPUTemp})
		}

		points++
	}

	// 再保险：若因起点被推迟导致超过 10 组，只保留最后 10 组
	trim := func(s [][]float64) [][]float64 {
		if len(s) > maxPoints {
			return s[len(s)-maxPoints:]
		}
		return s
	}
	cpuSeries = trim(cpuSeries)
	memSeries = trim(memSeries)
	tempSeries = trim(tempSeries)

	return
}

// node_levels.go 节点指标列表（GET /api/v2/observe/metrics/nodes）：原始读数 + 派生值 + 各读数的等级
package query

import (
	"context"

	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/model_v3/metrics"
)

// ListNodeMetricsViews 节点指标附上派生值与等级（阈值见 metrics_thresholds.go）。
// 集群还没有快照时返回 model.ErrSnapshotNotReady。
func (q *OTelQuery) ListNodeMetricsViews(ctx context.Context, clusterID string) ([]model.NodeMetricsView, error) {
	otel, err := q.GetOTelSnapshot(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	if otel == nil || otel.MetricsNodes == nil {
		return nil, model.ErrSnapshotNotReady
	}
	views := make([]model.NodeMetricsView, 0, len(otel.MetricsNodes))
	for i := range otel.MetricsNodes {
		views = append(views, nodeMetricsView(&otel.MetricsNodes[i]))
	}
	return views, nil
}

// nodeMetricsView 单节点：采集项没采到的读数等级为 unknown，⛔ 不按零值判成正常
func nodeMetricsView(nm *metrics.NodeMetrics) model.NodeMetricsView {
	un := unavailableSections(nm)
	view := model.NodeMetricsView{NodeMetrics: *nm}
	if i := nm.PrimaryDiskIndex(); i >= 0 {
		view.PrimaryDiskIndex = &i
	}

	lv := model.NodeLevels{
		CPU: model.HardwareUnknown, CPULoad: model.HardwareUnknown, IOWait: model.HardwareUnknown,
		Memory: model.HardwareUnknown, Swap: model.HardwareUnknown, OOMKill: model.HardwareUnknown,
		Disks: make([]model.NodeDiskLevels, len(nm.Disks)),
		PSI: model.NodePSILevels{
			CPUSome: model.HardwareUnknown, MemSome: model.HardwareUnknown, MemFull: model.HardwareUnknown,
			IOSome: model.HardwareUnknown, IOFull: model.HardwareUnknown,
		},
		VMStat: model.NodeVMStatLevels{
			MajorFaults: model.HardwareUnknown, SwapIn: model.HardwareUnknown, SwapOut: model.HardwareUnknown,
		},
		FileFD: model.HardwareUnknown, Conntrack: model.HardwareUnknown,
		ProcsBlocked: model.HardwareUnknown, TimeSync: model.HardwareUnknown, Entropy: model.HardwareUnknown,
	}
	if !un[metrics.SectionCPU] {
		lv.CPU = model.ResourceLevel(nm.CPU.UsagePct)
		lv.IOWait = model.WarnWhen(nm.CPU.IOWaitPct > iowaitWarnPct)
		if nm.CPU.Cores > 0 {
			perCore := hwRound(nm.CPU.Load1/float64(nm.CPU.Cores), 2)
			view.CPULoadPerCore = &perCore
			lv.CPULoad = model.LevelByThreshold(perCore, cpuLoadWarnPerCore, cpuLoadCritPerCore)
		}
	}
	if !un[metrics.SectionMemory] {
		lv.Memory = model.ResourceLevel(nm.Memory.UsagePct)
		lv.Swap = model.ResourceLevel(nm.Memory.SwapUsagePct)
		// OOM 是累计次数：发生过就留痕，使用率回落后这是唯一的现场
		lv.OOMKill = model.CritWhen(nm.Memory.OOMKillTotal > 0)
	}
	for i := range nm.Disks {
		if un[metrics.SectionDisks] {
			lv.Disks[i] = model.NodeDiskLevels{Usage: model.HardwareUnknown, Inode: model.HardwareUnknown, IOUtil: model.HardwareUnknown}
			continue
		}
		d := &nm.Disks[i]
		lv.Disks[i] = model.NodeDiskLevels{
			Usage:  model.ResourceLevel(d.UsagePct),
			Inode:  model.ResourceLevel(d.InodeUsagePct),
			IOUtil: model.ResourceLevel(d.IOUtilPct),
		}
	}
	if !un[metrics.SectionPSI] {
		psi := func(v float64) model.HardwareStatus { return model.LevelByThreshold(v, psiWarnPct, psiCritPct) }
		lv.PSI = model.NodePSILevels{
			CPUSome: psi(nm.PSI.CPUSomePct), MemSome: psi(nm.PSI.MemSomePct), MemFull: psi(nm.PSI.MemFullPct),
			IOSome: psi(nm.PSI.IOSomePct), IOFull: psi(nm.PSI.IOFullPct),
		}
	}
	if !un[metrics.SectionVMStat] {
		// 有换页就值得看一眼（K8s 节点默认不开 swap）
		lv.VMStat = model.NodeVMStatLevels{
			MajorFaults: model.CritWhen(nm.VMStat.PgMajFaultPerSec > majorFaultCritPerSec),
			SwapIn:      model.WarnWhen(nm.VMStat.PswpInPerSec > 0),
			SwapOut:     model.WarnWhen(nm.VMStat.PswpOutPerSec > 0),
		}
	}
	if !un[metrics.SectionSystem] {
		systemLevels(&nm.System, &view, &lv)
	}
	view.Levels = lv
	return view
}

// systemLevels 文件句柄 / 连接跟踪表 / D 状态进程 / 校时 / 熵
func systemLevels(sys *metrics.NodeSystem, view *model.NodeMetricsView, lv *model.NodeLevels) {
	switch {
	case sys.FilefdUnlimited:
		lv.FileFD = model.HardwareGood
	case sys.FilefdMax > 0:
		pct := hwRound(float64(sys.FilefdAllocated)/float64(sys.FilefdMax)*100, 2)
		view.FileFDUsagePct = &pct
		lv.FileFD = model.ResourceLevel(pct)
	}
	if sys.ConntrackLimit > 0 {
		pct := hwRound(float64(sys.ConntrackEntries)/float64(sys.ConntrackLimit)*100, 2)
		view.ConntrackUsagePct = &pct
		lv.Conntrack = model.ResourceLevel(pct)
	}
	// D 状态进程：进程卡在不可中断 IO 上，是磁盘出问题的早期信号
	lv.ProcsBlocked = model.WarnWhen(sys.ProcsBlocked > 0)
	// 时钟漂移会让日志与 trace 的时间线对不上
	lv.TimeSync = model.CritWhen(!sys.TimeSynced)
	lv.Entropy = model.CritWhen(sys.EntropyBits < entropyLowBits)
}

// unavailableSections Agent 声明采集失败的 section
func unavailableSections(nm *metrics.NodeMetrics) map[string]bool {
	un := make(map[string]bool, len(nm.Unavailable))
	for _, sec := range nm.Unavailable {
		un[sec] = true
	}
	return un
}

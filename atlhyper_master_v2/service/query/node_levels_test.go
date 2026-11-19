package query

import (
	"context"
	"errors"
	"testing"

	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/model_v3/cluster"
	"AtlHyper/model_v3/metrics"
)

func nodeViews(t *testing.T, nodes ...metrics.NodeMetrics) []model.NodeMetricsView {
	t.Helper()
	store := &mockStoreForOverview{snapshots: map[string]*cluster.ClusterSnapshot{
		"c1": {OTel: &cluster.OTelSnapshot{MetricsNodes: nodes}},
	}}
	views, err := (&OTelQuery{store: store}).ListNodeMetricsViews(context.Background(), "c1")
	if err != nil {
		t.Fatal(err)
	}
	if len(views) != len(nodes) {
		t.Fatalf("views = %d, want %d", len(views), len(nodes))
	}
	return views
}

// 各读数的等级由后端判定（阈值只在 metrics_thresholds.go），随数据下发；前端只按等级上色
func TestListNodeMetricsViews_Levels(t *testing.T) {
	v := nodeViews(t, metrics.NodeMetrics{
		NodeName: "busy",
		CPU:      metrics.NodeCPU{UsagePct: 85, IOWaitPct: 12, Load1: 6, Cores: 4},
		Memory:   metrics.NodeMemory{UsagePct: 40, SwapUsagePct: 95, OOMKillTotal: 3},
		PSI:      metrics.NodePSI{CPUSomePct: 53, IOFullPct: 85},
		VMStat:   metrics.NodeVMStat{PgMajFaultPerSec: 150, PswpInPerSec: 2},
		Disks: []metrics.NodeDisk{
			{MountPoint: "/boot/firmware", UsagePct: 50, TotalBytes: 512 << 20},
			{MountPoint: "/", UsagePct: 20, InodeUsagePct: 92, IOUtilPct: 81, TotalBytes: 200 << 30},
		},
		System: metrics.NodeSystem{
			FilefdAllocated: 9, FilefdMax: 10,
			ConntrackEntries: 1, ConntrackLimit: 100,
			ProcsBlocked: 2, TimeSynced: false, EntropyBits: 128,
		},
	})[0]

	lv := v.Levels
	if lv.CPU != model.HardwareWarn || lv.Memory != model.HardwareGood || lv.Swap != model.HardwareCrit {
		t.Errorf("cpu/memory/swap = %s/%s/%s, want warn/good/crit", lv.CPU, lv.Memory, lv.Swap)
	}
	if v.CPULoadPerCore == nil || *v.CPULoadPerCore != 1.5 || lv.CPULoad != model.HardwareCrit || lv.IOWait != model.HardwareWarn {
		t.Errorf("每核负载 = %v / %s，iowait = %s, want 1.5 / crit，warn", v.CPULoadPerCore, lv.CPULoad, lv.IOWait)
	}
	if lv.OOMKill != model.HardwareCrit {
		t.Errorf("OOM 发生过应为 crit，实际 %s", lv.OOMKill)
	}
	if lv.PSI.CPUSome != model.HardwareWarn || lv.PSI.IOFull != model.HardwareCrit || lv.PSI.MemSome != model.HardwareGood {
		t.Errorf("psi = %+v, want cpuSome warn / ioFull crit / memSome good", lv.PSI)
	}
	if lv.VMStat.MajorFaults != model.HardwareCrit || lv.VMStat.SwapIn != model.HardwareWarn || lv.VMStat.SwapOut != model.HardwareGood {
		t.Errorf("vmstat = %+v, want majorFaults crit / swapIn warn / swapOut good", lv.VMStat)
	}
	if len(lv.Disks) != 2 {
		t.Fatalf("disks levels = %d, want 2（与 disks 一一对应）", len(lv.Disks))
	}
	if root := lv.Disks[1]; root.Usage != model.HardwareGood || root.Inode != model.HardwareCrit || root.IOUtil != model.HardwareWarn {
		t.Errorf("根分区 = %+v, want usage good（容量 20%%）/ inode crit / ioUtil warn", root)
	}
	if v.PrimaryDiskIndex == nil || *v.PrimaryDiskIndex != 1 {
		t.Errorf("primaryDiskIndex = %v, want 1（根分区）", v.PrimaryDiskIndex)
	}
	if v.FileFDUsagePct == nil || *v.FileFDUsagePct != 90 || lv.FileFD != model.HardwareCrit {
		t.Errorf("fileFd = %v / %s, want 90 / crit", v.FileFDUsagePct, lv.FileFD)
	}
	if v.ConntrackUsagePct == nil || *v.ConntrackUsagePct != 1 || lv.Conntrack != model.HardwareGood {
		t.Errorf("conntrack = %v / %s, want 1 / good", v.ConntrackUsagePct, lv.Conntrack)
	}
	if lv.ProcsBlocked != model.HardwareWarn || lv.TimeSync != model.HardwareCrit || lv.Entropy != model.HardwareCrit {
		t.Errorf("blocked/timeSync/entropy = %s/%s/%s, want warn/crit/crit", lv.ProcsBlocked, lv.TimeSync, lv.Entropy)
	}
	if v.NodeName != "busy" {
		t.Error("原有字段应原样保留（内嵌）")
	}
}

// 没采到的采集项：等级是 unknown，⛔ 不按零值判成正常；没有上限的比率为 null
func TestListNodeMetricsViews_UnavailableIsUnknown(t *testing.T) {
	v := nodeViews(t, metrics.NodeMetrics{
		NodeName:    "partial",
		Disks:       []metrics.NodeDisk{{MountPoint: "/"}},
		Unavailable: []string{metrics.SectionCPU, metrics.SectionDisks, metrics.SectionSystem, metrics.SectionPSI, metrics.SectionVMStat},
	}, metrics.NodeMetrics{
		NodeName: "unlimited",
		System:   metrics.NodeSystem{FilefdUnlimited: true, TimeSynced: true, EntropyBits: 256},
	})

	p := v[0].Levels
	if p.CPU != model.HardwareUnknown || p.Disks[0].Usage != model.HardwareUnknown || p.FileFD != model.HardwareUnknown || p.Entropy != model.HardwareUnknown {
		t.Errorf("没采到的项应为 unknown: %+v", p)
	}
	if p.CPULoad != model.HardwareUnknown || p.PSI.IOFull != model.HardwareUnknown || p.VMStat.MajorFaults != model.HardwareUnknown || v[0].CPULoadPerCore != nil {
		t.Errorf("CPU / PSI / vmstat 没采到应为 unknown: cpuLoad=%s psi=%+v vmstat=%+v", p.CPULoad, p.PSI, p.VMStat)
	}
	if p.Memory != model.HardwareGood {
		t.Errorf("内存采到了（0%%）应为 good，实际 %s", p.Memory)
	}

	u := v[1]
	if u.FileFDUsagePct != nil || u.Levels.FileFD != model.HardwareGood {
		t.Errorf("文件句柄无上限: pct=%v level=%s, want null / good", u.FileFDUsagePct, u.Levels.FileFD)
	}
	if u.ConntrackUsagePct != nil || u.Levels.Conntrack != model.HardwareUnknown {
		t.Errorf("取不到连接跟踪上限: pct=%v level=%s, want null / unknown", u.ConntrackUsagePct, u.Levels.Conntrack)
	}
	if u.PrimaryDiskIndex != nil {
		t.Errorf("没有盘时 primaryDiskIndex 应为 null，实际 %d", *u.PrimaryDiskIndex)
	}
}

func TestListNodeMetricsViews_NoSnapshot(t *testing.T) {
	svc := &OTelQuery{store: &mockStoreForOverview{snapshots: map[string]*cluster.ClusterSnapshot{}}}
	if _, err := svc.ListNodeMetricsViews(context.Background(), "nope"); !errors.Is(err, model.ErrSnapshotNotReady) {
		t.Fatalf("没有快照应返回 ErrSnapshotNotReady，实际 %v", err)
	}
}

// 硬件速览的告警节点数 = 整行结论为 warn / crit 的节点（含温度等硬件项，不只是使用率）
func TestHardwareSummary_WarningNodes(t *testing.T) {
	rows := []model.HardwareRow{
		{NodeName: "a", Overall: model.HardwareGood},
		{NodeName: "b", Overall: model.HardwareWarn},
		{NodeName: "c", Overall: model.HardwareCrit},
		{NodeName: "d", Overall: model.HardwareUnknown},
	}
	if got := buildHardwareSummary(nil, rows).WarningNodes; got != 2 {
		t.Errorf("warningNodes = %d, want 2", got)
	}
}

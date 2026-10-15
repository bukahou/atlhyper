package query

import (
	"testing"

	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/model_v3/metrics"
)

// 机械盘与 NVMe 的健康线相差 30°C 以上：更热的那块不一定更危险。
// 格子必须先按状态取最差，同档才比温度 —— 否则超线的机械盘会藏在正常的 NVMe 后面。
func hddWarnBehindHotterNVMe() []metrics.TempSensor {
	return []metrics.TempSensor{
		{Chip: "nvme_nvme0", ChipName: "nvme", Sensor: "temp1", CurrentC: 58, MaxC: 84, CritC: 89},
		{Chip: "target0:0:0_0:0:0:0", ChipName: "drivetemp", Sensor: "temp1", CurrentC: 55, MaxC: 50, CritC: 60},
	}
}

func TestTempCell_WorstStatusBeatsHotter(t *testing.T) {
	th := thresholdsFor(metrics.ProfileDesk)
	cell := tempCell(hddWarnBehindHotterNVMe(), metrics.TempClassDisk, th.disk)
	if cell == nil {
		t.Fatal("expected disk cell")
	}
	if cell.Status != model.HardwareWarn || cell.Value != 55 || cell.Label != "drivetemp" {
		t.Errorf("应取超线的机械盘 55°C/warn，得到 %+v", cell)
	}
}

func TestTempCell_SameStatusPicksHotter(t *testing.T) {
	sensors := []metrics.TempSensor{
		{Chip: "nvme_nvme0", ChipName: "nvme", Sensor: "temp1", CurrentC: 41, MaxC: 84, CritC: 89},
		{Chip: "nvme_nvme1", ChipName: "nvme", Sensor: "temp1", CurrentC: 47, MaxC: 84, CritC: 89},
	}
	cell := tempCell(sensors, metrics.TempClassDisk, thresholdsFor(metrics.ProfileDesk).disk)
	if cell == nil || cell.Value != 47 || cell.Label != "nvme1" {
		t.Errorf("同档应取更热的 nvme1 47°C，得到 %+v", cell)
	}
}

func TestBuildHardwareRow_HiddenHDDWarnReachesOverall(t *testing.T) {
	nm := &metrics.NodeMetrics{
		NodeName: "hdd-node", HardwareProfile: metrics.ProfileDesk,
		Temperature: metrics.NodeTemperature{Sensors: hddWarnBehindHotterNVMe()},
	}
	row := buildHardwareRow(nm)
	if row.DiskTemp == nil || row.DiskTemp.Status != model.HardwareWarn {
		t.Errorf("diskTemp = %+v，期望 warn", row.DiskTemp)
	}
	if row.Overall != model.HardwareWarn {
		t.Errorf("overall = %q，期望 warn", row.Overall)
	}
}

// 速览 tile 跨节点、跨类别比较：CPU 更热但正常，另一台的机械盘超线 → tile 必须指向机械盘。
func TestBuildHardwareSummary_WorstStatusBeatsHotter(t *testing.T) {
	nodes := []metrics.NodeMetrics{
		{NodeName: "desk-zero", HardwareProfile: metrics.ProfileDesk,
			Temperature: metrics.NodeTemperature{Sensors: []metrics.TempSensor{
				{Chip: "platform_coretemp_0", ChipName: "coretemp", Sensor: "temp1", CurrentC: 74, MaxC: 80, CritC: 100},
			}}},
		{NodeName: "hdd-node", HardwareProfile: metrics.ProfileDesk,
			Temperature: metrics.NodeTemperature{Sensors: hddWarnBehindHotterNVMe()}},
	}
	rows := make([]model.HardwareRow, len(nodes))
	for i := range nodes {
		rows[i] = buildHardwareRow(&nodes[i])
	}
	s := buildHardwareSummary(nodes, rows)
	if s.MaxTemp == nil || s.MaxTemp.NodeName != "hdd-node" || s.MaxTemp.Value != 55 || s.MaxTemp.Status != model.HardwareWarn {
		t.Errorf("maxTemp = %+v，期望 hdd-node 机械盘 55°C/warn", s.MaxTemp)
	}
	if s.MaxDiskTemp == nil || s.MaxDiskTemp.Value != 55 || s.MaxDiskTemp.Status != model.HardwareWarn {
		t.Errorf("maxDiskTemp = %+v，期望机械盘 55°C/warn", s.MaxDiskTemp)
	}
}

// 只自报 max、不报 crit 时，画像的 crit 可能低于自报 max（实测 r8169 网卡 max=120，其他档 crit=95）。
// 与「只报 crit → max=crit×0.9」对称：crit 至少为 max÷0.9，保证 warn 区间存在。
func TestResolveTempLimits_MaxOnlyKeepsWarnBand(t *testing.T) {
	cases := []struct {
		name               string
		sensorMax, sensorC float64
		fb                 tempThreshold
		wantMax, wantCrit  float64
	}{
		{"自报 max 高于画像 crit", 120, 0, tempThreshold{max: 85, crit: 95}, 120, 133.33},
		{"自报 max 低于画像 crit：不变", 70, 0, tempThreshold{max: 85, crit: 95}, 70, 95},
		{"两者都报：不变", 84, 89, tempThreshold{max: 70, crit: 80}, 84, 89},
		{"只报 crit：不变", 0, 100, tempThreshold{max: 85, crit: 95}, 90, 100},
		{"都不报：画像", 0, 0, tempThreshold{max: 70, crit: 80}, 70, 80},
	}
	for _, c := range cases {
		gotMax, gotCrit := resolveTempLimits(c.sensorMax, c.sensorC, c.fb)
		if gotMax != c.wantMax || gotCrit != c.wantCrit {
			t.Errorf("%s: got %v/%v, want %v/%v", c.name, gotMax, gotCrit, c.wantMax, c.wantCrit)
		}
		if gotCrit <= gotMax {
			t.Errorf("%s: crit %v 必须高于 max %v", c.name, gotCrit, gotMax)
		}
	}
}

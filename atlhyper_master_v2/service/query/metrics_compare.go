// metrics_compare.go 节点对比表（GET /api/v2/observe/metrics/compare）
//
// 硬件矩阵回答「这台机器的传感器怎么样」，对比表回答「七台里哪台不一样」。
// 因此这里不重新判定硬件 —— 直接复用 buildHardwareRow 的结论，
// 只补上四大资源那几列。同一个读数在两个页面显示不同颜色是最难排查的 UI bug。
package query

import (
	"context"
	"fmt"

	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/model_v3/metrics"
)

// compareColumns 列顺序：硬件在前（会烧板子的先看），资源在后。
// 前端按这个数组渲染表头，不做重排。
var compareColumns = []string{
	"cpuTemp", "diskTemp", "undervolt", "freqRatio", "diskAwait",
	"diskUsage", "cpu", "psiCpu", "mem", "psiMem", "netErr",
}

// 各列的告警线都在 metrics_thresholds.go。

// GetNodeComparison 产出节点横向对比表
func (q *OTelQuery) GetNodeComparison(ctx context.Context, clusterID string) (*model.NodeComparisonResponse, error) {
	resp := &model.NodeComparisonResponse{Columns: compareColumns, Rows: []model.CompareRow{}}

	otel, err := q.GetOTelSnapshot(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	if otel == nil {
		return resp, nil
	}

	for i := range otel.MetricsNodes {
		nm := &otel.MetricsNodes[i]
		hwRow := buildHardwareRow(nm) // 复用硬件判定，不重算
		cells := compareCells(nm, hwRow)
		// 整行结论 = 硬件矩阵那一行的结论 + 对比表多出来的列（PSI / 网卡错误）
		overall := hwRow.Overall
		for _, c := range cells {
			overall = worstStatus(overall, c.Status)
		}
		resp.Rows = append(resp.Rows, model.CompareRow{
			NodeName: nm.NodeName,
			Profile:  hwRow.Profile,
			Cells:    cells,
			Overall:  overall,
		})
	}
	return resp, nil
}

// compareCells 每列都必须有键 —— 缺信号用 value=nil，省略键会让表格错位
func compareCells(nm *metrics.NodeMetrics, hw model.HardwareRow) map[string]model.CompareCell {
	cells := make(map[string]model.CompareCell, len(compareColumns))

	cells["cpuTemp"] = tempCompareCell(hw.CPUTemp, "°C")
	cells["diskTemp"] = tempCompareCell(hw.DiskTemp, "°C")

	// 欠压用 0/1 而不是文字：文案属于前端 i18n，后端只给语义
	if hw.Undervolt != nil {
		alarm := 0.0
		if hw.Undervolt.Alarm {
			alarm = 1
		}
		cells["undervolt"] = model.CompareCell{Value: &alarm, Status: hw.Undervolt.Status}
	} else {
		cells["undervolt"] = model.CompareCell{Status: model.HardwareUnknown}
	}

	if hw.CPUFreq != nil {
		v := hw.CPUFreq.RatioPct
		cells["freqRatio"] = model.CompareCell{Value: &v, Text: fmt.Sprintf("%.0f%%", v), Status: hw.CPUFreq.Status}
	} else {
		cells["freqRatio"] = model.CompareCell{Status: model.HardwareUnknown}
	}

	if hw.DiskAwait != nil {
		v := hw.DiskAwait.ValueMs
		cells["diskAwait"] = model.CompareCell{Value: &v, Text: fmt.Sprintf("%.1f ms", v), Status: hw.DiskAwait.Status}
	} else {
		cells["diskAwait"] = model.CompareCell{Status: model.HardwareUnknown}
	}

	// 资源列：使用率沿用硬件矩阵的判定（同一份阈值，采集项没采到时为无数据）
	cells["diskUsage"] = usageCompareCell(hw.DiskUsage)
	cells["cpu"] = usageCompareCell(hw.CPUUsage)
	cells["mem"] = usageCompareCell(hw.MemUsage)
	un := unavailableSections(nm)
	if un[metrics.SectionPSI] {
		cells["psiCpu"] = model.CompareCell{Status: model.HardwareUnknown}
		cells["psiMem"] = model.CompareCell{Status: model.HardwareUnknown}
	} else {
		cells["psiCpu"] = pctCell(nm.PSI.CPUSomePct, psiWarnPct, psiCritPct)
		cells["psiMem"] = pctCell(nm.PSI.MemSomePct, psiWarnPct, psiCritPct)
	}
	if un[metrics.SectionNetworks] {
		cells["netErr"] = model.CompareCell{Status: model.HardwareUnknown}
	} else {
		cells["netErr"] = netErrCell(nm.Networks)
	}

	return cells
}

func tempCompareCell(cell *model.HardwareTempCell, unit string) model.CompareCell {
	if cell == nil {
		return model.CompareCell{Status: model.HardwareUnknown}
	}
	v := cell.Value
	return model.CompareCell{Value: &v, Text: fmt.Sprintf("%.1f%s", v, unit), Status: cell.Status}
}

func pctCell(v, warn, crit float64) model.CompareCell {
	value := hwRound(v, 1)
	return model.CompareCell{Value: &value, Text: fmt.Sprintf("%.1f%%", value), Status: model.LevelByThreshold(value, warn, crit)}
}

// usageCompareCell 硬件矩阵的使用率格 → 对比表的格；没有读数为无数据
func usageCompareCell(cell *model.HardwareUsageCell) model.CompareCell {
	if cell == nil {
		return model.CompareCell{Status: model.HardwareUnknown}
	}
	v := cell.Value
	return model.CompareCell{Value: &v, Text: fmt.Sprintf("%.1f%%", v), Status: cell.Status}
}

// netErrCell 只统计真错误（err），不含 drop。
//
// 虚拟接口（lxc*/cilium_*）的 drop 常态非零 —— 组播与广播被内核丢掉是正常的，
// 实测 7 台全部 0.50/s。把 drop 算进来的结果是整列全黄，等于没有信号。
// drop 仍然在节点卡的 NetworkCard 里按接口列出。
func netErrCell(nets []metrics.NodeNetwork) model.CompareCell {
	var total float64
	for _, n := range nets {
		total += n.RxErrPerSec + n.TxErrPerSec
	}
	total = hwRound(total, 2)
	cell := model.CompareCell{Value: &total, Text: fmt.Sprintf("%.2f/s", total), Status: model.HardwareGood}
	if total > 0 {
		cell.Status = model.HardwareWarn
	}
	return cell
}

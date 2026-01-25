package node

import (
	"context"

	"AtlHyper/atlhyper_agent/sdk"
	modelnode "AtlHyper/model/k8s"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metricsv1beta1 "k8s.io/metrics/pkg/apis/metrics/v1beta1"
)

// 批量拉取 Node 指标：key = nodeName
func fetchNodeMetricsMap(ctx context.Context) map[string]*metricsv1beta1.NodeMetrics {
	out := map[string]*metricsv1beta1.NodeMetrics{}
	if !sdk.Get().HasMetricsServer() {
		return out
	}
	mc := sdk.Get().MetricsClient()
	list, err := mc.MetricsV1beta1().NodeMetricses().List(ctx, metav1.ListOptions{})
	if err != nil || list == nil {
		return out
	}
	for i := range list.Items {
		nm := &list.Items[i]
		out[nm.Name] = nm
	}
	return out
}

// 在骨架上填充 metrics（nm 可为 nil；podsUsed 仅用于 Pod 槽位统计）
func attachMetrics(dst *modelnode.Node, nm *metricsv1beta1.NodeMetrics, podsUsed int) {
	// 资源基线：优先 allocatable，回退 capacity
	var (
		cpuAllocCores  = quantityCores(dst.Allocatable.CPU, dst.Capacity.CPU)
		memAllocBytes  = quantityBytes(dst.Allocatable.Memory, dst.Capacity.Memory)
		podCap         = quantityInt(dst.Allocatable.Pods, dst.Capacity.Pods)
		cpuCapCores    = quantityCores(dst.Capacity.CPU, "")
		memCapBytes    = quantityBytes(dst.Capacity.Memory, "")
	)

	// usage：来自 metrics（若缺失则为 0）
	var cpuUsageCores float64
	var memUsageBytes float64
	if nm != nil {
		if q := nm.Usage.Cpu(); q != nil {
			cpuUsageCores = q.AsApproximateFloat64()
		}
		if q := nm.Usage.Memory(); q != nil {
			memUsageBytes = float64(q.Value())
		}
	}

	// 组装字符串表现
	cpuUsageStr := resource.NewMilliQuantity(int64(cpuUsageCores*1000), resource.DecimalSI).String() // m
	memUsageStr := resource.NewQuantity(int64(memUsageBytes), resource.BinarySI).String()

	allocCPU := normalizeCPU(dst.Allocatable.CPU, dst.Capacity.CPU)
	allocMem := normalizeBytes(dst.Allocatable.Memory, dst.Capacity.Memory)
	capCPU := normalizeCPU(dst.Capacity.CPU, "")
	capMem := normalizeBytes(dst.Capacity.Memory, "")

	cpuPct, _ := calcPct(cpuUsageCores, cpuAllocCores, cpuCapCores)
	memPct, _ := calcPct(memUsageBytes, memAllocBytes, memCapBytes)

	// Pods metric
	if podCap <= 0 {
		// 回退：尽量从 capacity.pods 解析字符串
		podCap = quantityInt(dst.Capacity.Pods, "")
	}
	var podPct float64
	if podCap > 0 && podsUsed >= 0 {
		podPct = float64(int((float64(podsUsed)/float64(podCap)*100)*10+0.5)) / 10
	}

	dst.Metrics = &modelnode.NodeMetrics{
		CPU:    modelnode.NodeResourceMetric{Usage: cpuUsageStr, Allocatable: allocCPU, Capacity: capCPU, UtilPct: cpuPct},
		Memory: modelnode.NodeResourceMetric{Usage: memUsageStr, Allocatable: allocMem, Capacity: capMem, UtilPct: memPct},
		Pods:   modelnode.PodCountMetric{Used: podsUsed, Capacity: podCap, UtilPct: podPct},
		Pressure: buildPressureFlags(dst.Conditions),
	}
}

// UtilPct: usage / (alloc|cap) * 100，限制在 [0,100]，保留 1 位小数
func calcPct(usage, alloc, cap float64) (float64, bool) {
	den := 0.0
	switch {
	case alloc > 0:
		den = alloc
	case cap > 0:
		den = cap
	default:
		return 0, false
	}
	p := usage / den * 100
	if p < 0 {
		p = 0
	} else if p > 100 {
		p = 100
	}
	return float64(int(p*10+0.5)) / 10, true
}

func buildPressureFlags(conds []modelnode.NodeCondition) modelnode.PressureFlags {
	var pf modelnode.PressureFlags
	for _, c := range conds {
		if c.Status != "True" {
			continue
		}
		switch c.Type {
		case string(corev1.NodeMemoryPressure):
			pf.MemoryPressure = true
		case string(corev1.NodeDiskPressure):
			pf.DiskPressure = true
		case string(corev1.NodePIDPressure):
			pf.PIDPressure = true
		case string(corev1.NodeNetworkUnavailable):
			pf.NetworkUnavailable = true
		}
	}
	return pf
}

package pod

import (
	"AtlHyper/atlhyper_agent/utils"
	modelpod "AtlHyper/model/pod"
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metricsv1beta1 "k8s.io/metrics/pkg/apis/metrics/v1beta1"
)

// fetchPodMetricsMap —— 批量拉取某 ns（或全局）Pod 指标，key=ns/name
// buildPodMetrics —— 汇总 usage & limits/requests，计算 UtilPct
func buildPodMetrics(pm *metricsv1beta1.PodMetrics, specContainers []corev1.Container) *modelpod.PodMetrics {
	// 1) usage 汇总
	var cpuUsageCores float64
	var memUsageBytes float64
	for i := range pm.Containers {
		u := pm.Containers[i].Usage
		if cq := u.Cpu(); cq != nil {
			cpuUsageCores += cq.AsApproximateFloat64() // cores
		}
		if mq := u.Memory(); mq != nil {
			memUsageBytes += float64(mq.Value()) // bytes
		}
	}

	// 2) limits/requests 汇总（来自 spec）
	var cpuLimitCores, cpuReqCores float64
	var memLimitBytes, memReqBytes float64
	for _, c := range specContainers {
		if q := c.Resources.Limits.Cpu(); q != nil {
			cpuLimitCores += q.AsApproximateFloat64()
		}
		if q := c.Resources.Requests.Cpu(); q != nil {
			cpuReqCores += q.AsApproximateFloat64()
		}
		if q := c.Resources.Limits.Memory(); q != nil {
			memLimitBytes += float64(q.Value())
		}
		if q := c.Resources.Requests.Memory(); q != nil {
			memReqBytes += float64(q.Value())
		}
	}

	// 3) 字符串格式化
	cpuUsageStr := resource.NewQuantity(int64(cpuUsageCores*1000), resource.DecimalSI).String() // mCPU
	cpuLimitStr := ""
	if cpuLimitCores > 0 {
		cpuLimitStr = resource.NewQuantity(int64(cpuLimitCores*1000), resource.DecimalSI).String()
	}
	memUsageStr := resource.NewQuantity(int64(memUsageBytes), resource.BinarySI).String()
	memLimitStr := ""
	if memLimitBytes > 0 {
		memLimitStr = resource.NewQuantity(int64(memLimitBytes), resource.BinarySI).String()
	}

	// 4) 使用率
	cpuPct, _ := calcPct(cpuUsageCores, cpuLimitCores, cpuReqCores)
	memPct, _ := calcPct(memUsageBytes, memLimitBytes, memReqBytes)

	return &modelpod.PodMetrics{
		CPU:    modelpod.ResourceMetric{Usage: cpuUsageStr, Limit: cpuLimitStr, UtilPct: cpuPct},
		Memory: modelpod.ResourceMetric{Usage: memUsageStr, Limit: memLimitStr, UtilPct: memPct},
	}
}


// calcPct: usage / (limit|request) * 100，限制在 [0,100]，保留一位小数
func calcPct(usage, limit, request float64) (float64, bool) {
	den := 0.0
	switch {
	case limit > 0:
		den = limit
	case request > 0:
		den = request
	default:
		return 0, false
	}
	p := usage / den * 100
	if p < 0 {
		p = 0
	} else if p > 100 {
		p = 100
	}
	// 保留一位小数（不依赖 math 包）
	return float64(int(p*10+0.5)) / 10, true
}


func fetchPodMetricsMap(ctx context.Context, namespace string) map[string]*metricsv1beta1.PodMetrics {
	out := map[string]*metricsv1beta1.PodMetrics{}
	if !utils.HasMetricsServer() {
		return out
	}
	mc := utils.GetMetricsClient()

	var (
		list *metricsv1beta1.PodMetricsList
		err  error
	)
	if namespace == "" {
		list, err = mc.MetricsV1beta1().PodMetricses(metav1.NamespaceAll).List(ctx, metav1.ListOptions{})
	} else {
		list, err = mc.MetricsV1beta1().PodMetricses(namespace).List(ctx, metav1.ListOptions{})
	}
	if err != nil || list == nil {
		return out
	}

	for i := range list.Items {
		pm := &list.Items[i]
		out[pm.Namespace+"/"+pm.Name] = pm
	}
	return out
}
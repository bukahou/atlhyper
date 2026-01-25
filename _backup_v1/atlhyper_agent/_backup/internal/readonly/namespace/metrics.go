// internal/readonly/namespace/metrics.go
package namespace

import (
	"context"

	modelns "AtlHyper/model/k8s"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"AtlHyper/atlhyper_agent/utils"
)

// buildNamespaceMetrics —— 基于 Pod 列表 + metrics-server 的 PodMetrics 聚合出每个 NS 的 CPU/内存
// - 若未部署 metrics-server：返回空 map，不影响主流程
func buildNamespaceMetrics(ctx context.Context, pods []corev1.Pod) (map[string]*modelns.NamespaceMetrics, error) {
	out := map[string]*modelns.NamespaceMetrics{}
	if !utils.HasMetricsServer() {
		return out, nil
	}
	mc := utils.GetMetricsClient()

	// 1) 拉全量 PodMetrics
	pmList, err := mc.MetricsV1beta1().PodMetricses(metav1.NamespaceAll).List(ctx, metav1.ListOptions{})
	if err != nil || pmList == nil {
		return out, err
	}

	// 2) requests/limits：从 Pod 规格聚合
	// type agg struct {
	// 	cpuUsageCores float64
	// 	memUsageBytes float64

	// 	cpuReqCores float64
	// 	cpuLimCores float64
	// 	memReqBytes float64
	// 	memLimBytes float64
	// }
	nsAgg := map[string]*agg{}

	// 2.1 聚合 requests/limits（来自 Pod.Spec.Containers）
	for i := range pods {
		p := &pods[i]
		a := ensureAgg(nsAgg, p.Namespace)
		for _, c := range p.Spec.Containers {
			if q := c.Resources.Requests.Cpu(); q != nil {
				a.cpuReqCores += q.AsApproximateFloat64()
			}
			if q := c.Resources.Limits.Cpu(); q != nil {
				a.cpuLimCores += q.AsApproximateFloat64()
			}
			if q := c.Resources.Requests.Memory(); q != nil {
				a.memReqBytes += float64(q.Value())
			}
			if q := c.Resources.Limits.Memory(); q != nil {
				a.memLimBytes += float64(q.Value())
			}
		}
	}

	// 2.2 聚合 usage（来自 PodMetrics）
	for i := range pmList.Items {
		pm := &pmList.Items[i]
		a := ensureAgg(nsAgg, pm.Namespace)
		for _, c := range pm.Containers {
			u := c.Usage
			if q := u.Cpu(); q != nil {
				a.cpuUsageCores += q.AsApproximateFloat64()
			}
			if q := u.Memory(); q != nil {
				a.memUsageBytes += float64(q.Value())
			}
		}
	}

	// 3) 落到模型（字符串格式化 & 使用率）
	for ns, a := range nsAgg {
		// CPU
		cpuUsageStr := resource.NewQuantity(int64(a.cpuUsageCores*1000), resource.DecimalSI).String() // m
		cpuReqStr := ""
		cpuLimStr := ""
		if a.cpuReqCores > 0 {
			cpuReqStr = resource.NewQuantity(int64(a.cpuReqCores*1000), resource.DecimalSI).String()
		}
		if a.cpuLimCores > 0 {
			cpuLimStr = resource.NewQuantity(int64(a.cpuLimCores*1000), resource.DecimalSI).String()
		}
		cpuPct, basis := calcUtilPct(a.cpuUsageCores, a.cpuLimCores, a.cpuReqCores)

		// Memory
		memUsageStr := resource.NewQuantity(int64(a.memUsageBytes), resource.BinarySI).String()
		memReqStr := ""
		memLimStr := ""
		if a.memReqBytes > 0 {
			memReqStr = resource.NewQuantity(int64(a.memReqBytes), resource.BinarySI).String()
		}
		if a.memLimBytes > 0 {
			memLimStr = resource.NewQuantity(int64(a.memLimBytes), resource.BinarySI).String()
		}
		memPct, basisM := calcUtilPct(a.memUsageBytes, a.memLimBytes, a.memReqBytes)

		out[ns] = &modelns.NamespaceMetrics{
			CPU: modelns.ResourceAgg{
				Usage:     cpuUsageStr,
				Requests:  cpuReqStr,
				Limits:    cpuLimStr,
				UtilPct:   cpuPct,
				UtilBasis: basis,
			},
			Memory: modelns.ResourceAgg{
				Usage:     memUsageStr,
				Requests:  memReqStr,
				Limits:    memLimStr,
				UtilPct:   memPct,
				UtilBasis: basisM,
			},
		}
	}
	return out, nil
}


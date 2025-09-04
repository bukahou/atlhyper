package pod

import (
	"context"
	"fmt"

	"AtlHyper/atlhyper_agent/utils"
	modelpod "AtlHyper/model/pod"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ListPods —— ★ 唯一对外入口：全集群总查询
// 编排：一次性拉全量 corev1.Pod → 为每个 Pod 构建“骨架” → 批量获取 metrics 并就地填充
func ListPods(ctx context.Context) ([]modelpod.Pod, error) {
	// t0 := time.Now()

	cs := utils.GetCoreClient()

	// 1) 全集群 Pod 列表
	podList, err := cs.CoreV1().Pods(metav1.NamespaceAll).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list pods (cluster-wide) failed: %w", err)
	}
	// t1 := time.Now()

	n := len(podList.Items)
	out := make([]modelpod.Pod, n)
	keys := make([]string, n)                   // ns/name 对齐 metrics
	specIdx := make(map[string]interface{}, n)  // 保存 spec.containers，供 attachMetrics 使用

	// 2) 先构建“静态骨架”（summary/spec/containers/volumes/network）
	for i := range podList.Items {
		cp := &podList.Items[i]
		out[i] = buildSkeleton(cp) // 私有装配，不含 metrics
		k := cp.Namespace + "/" + cp.Name
		keys[i] = k
		specIdx[k] = cp.Spec.Containers // attachMetrics 需要
	}
	// t2 := time.Now()

	// 3) 批量拉取 metrics（无 metrics-server 时返回空，不影响主流程），并就地填充
	metricsMap := fetchPodMetricsMap(ctx, "")
	// t3 := time.Now()
	if len(metricsMap) > 0 {
		for i, k := range keys {
			if pm, ok := metricsMap[k]; ok {
				attachMetrics(&out[i], pm, specIdx[k])
			}
		}
	}
	// t4 := time.Now()

	// 统计 metrics 命中数
	hits := 0
	if len(metricsMap) > 0 {
		for _, k := range keys {
			if _, ok := metricsMap[k]; ok {
				hits++
			}
		}
	}

	return out, nil
}

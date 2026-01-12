// internal/readonly/namespace/list.go
package namespace

import (
	"context"
	"fmt"
	"log"

	"AtlHyper/atlhyper_agent/sdk"
	modelns "AtlHyper/model/k8s"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ListNamespaces —— ★ 对外唯一入口：全集群总查询（骨架 → 计数/配额/限制 → 指标 → 组装）
func ListNamespaces(ctx context.Context) ([]modelns.Namespace, error) {
	cs := sdk.Get().CoreClient()

	// t0 := time.Now()
	// 1) 全集群 Namespaces
	nsList, err := cs.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list namespaces failed: %w", err)
	}
	// t1 := time.Now()

	// 2) 全集群 Pods（供 counts ＆ metrics 使用；只拉一次避免重复 IO）
	podList, err := cs.CoreV1().Pods(metav1.NamespaceAll).List(ctx, metav1.ListOptions{})
	if err != nil {
		// 不致命：骨架仍可返回
		log.Printf("[namespace] warn: list pods for aggregation failed: %v", err)
	}
	// t2 := time.Now()

	// 3) 先构建骨架
	n := len(nsList.Items)
	out := make([]modelns.Namespace, n)
	keys := make([]string, n)
	for i := range nsList.Items {
		ns := &nsList.Items[i]
		out[i] = buildSkeleton(ns)
		keys[i] = ns.Name
	}
	// t3 := time.Now()

	// 4) 计数聚合（Deployments/Pods/...）
	var pods []corev1.Pod
	if podList != nil {
		pods = podList.Items
	}
	countsByNS := aggregateCounts(ctx, pods)
	// t4 := time.Now()

	// 5) 配额与限制
	quotasByNS, exceededNS, err := indexQuotasByNS(ctx, cs)
	if err != nil {
		log.Printf("[namespace] warn: index quotas failed: %v", err)
	}
	limitsByNS, err := indexLimitRangesByNS(ctx, cs)
	if err != nil {
		log.Printf("[namespace] warn: index limitRanges failed: %v", err)
	}
	// t5 := time.Now()

	// 6) 指标聚合（metrics-server 可选）
	metricsByNS, _ := buildNamespaceMetrics(ctx, pods) // 若无 metrics-server，返回空 map
	// t6 := time.Now()

	// 7) 组装附加块
	for i, name := range keys {
		if c, ok := countsByNS[name]; ok {
			out[i].Counts = c
		}
		if qs, ok := quotasByNS[name]; ok {
			out[i].Quotas = qs
		}
		if ls, ok := limitsByNS[name]; ok {
			out[i].LimitRanges = ls
		}
		if m, ok := metricsByNS[name]; ok {
			out[i].Metrics = m
		}
		attachBadges(&out[i], exceededNS[name])
	}
	// t7 := time.Now()

	// 8) 阶段耗时日志（微秒）
	// podsN := 0
	// if podList != nil {
	// 	podsN = len(podList.Items)
	// }
	// log.Printf("[namespace_list] count=%d pods=%d phases_us list_ns=%d list_pods=%d build=%d counts=%d quotas_limits=%d metrics=%d attach=%d total=%d",
	// 	n, podsN,
	// 	t1.Sub(t0).Microseconds(),
	// 	t2.Sub(t1).Microseconds(),
	// 	t3.Sub(t2).Microseconds(),
	// 	t4.Sub(t3).Microseconds(),
	// 	t5.Sub(t4).Microseconds(),
	// 	t6.Sub(t5).Microseconds(),
	// 	t7.Sub(t6).Microseconds(),
	// 	t7.Sub(t0).Microseconds(),
	// )

	return out, nil
}

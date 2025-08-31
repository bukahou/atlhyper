package node

import (
	"context"
	"fmt"
	"log"

	"AtlHyper/atlhyper_agent/utils"
	modelnode "AtlHyper/model/node"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ListNodes —— ★ 唯一对外入口：全集群 Node 查询（含 metrics & pods used）
func ListNodes(ctx context.Context) ([]modelnode.Node, error) {
	// t0 := time.Now()
	cs := utils.GetCoreClient()

	// 1) 拉 Node 列表（cluster-wide）
	nl, err := cs.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list nodes failed: %w", err)
	}
	// t1 := time.Now()

	n := len(nl.Items)
	out := make([]modelnode.Node, n)
	names := make([]string, n)

	// 2) 构建“静态骨架”（summary/spec/resources/addresses/info/conditions/taints/labels）
	for i := range nl.Items {
		cn := &nl.Items[i]
		out[i] = buildSkeleton(cn)
		names[i] = cn.Name
	}
	// t2 := time.Now()

	// 3) 统计每节点的 Pod 使用数（一次性拉全量 Pod 并按节点归并）
	podsPerNode, err := countPodsPerNode(ctx)
	if err != nil {
		// 不阻塞主流程，仅日志
		log.Printf("[node_list] warn count pods per node: %v", err)
	}
	// t3 := time.Now()

	// 4) 批量拉取 Node metrics，并就地填充
	metricsMap := fetchNodeMetricsMap(ctx)
	if len(metricsMap) > 0 {
		for i, name := range names {
			if nm, ok := metricsMap[name]; ok {
				attachMetrics(&out[i], nm, podsPerNode[name])
			} else {
				// 没有 metrics 也可用 podsPerNode 补齐 Pod 槽位信息
				attachMetrics(&out[i], nil, podsPerNode[name])
			}
		}
	} else {
		// 无 metrics-server，仅补 pods 槽位
		for i, name := range names {
			attachMetrics(&out[i], nil, podsPerNode[name])
		}
	}
	// t4 := time.Now()

	// // 分阶段耗时（微秒）
	// log.Printf("[node_list] count=%d phases_us list=%d build=%d pods_used=%d fetch_metrics=%d attach=%d total=%d",
	// 	n,
	// 	t1.Sub(t0).Microseconds(),
	// 	t2.Sub(t1).Microseconds(),
	// 	t3.Sub(t2).Microseconds(),
	// 	t4.Sub(t3).Microseconds(),
	// 	0, // attach 已计入上行（可细分则改为单独计时）
	// 	t4.Sub(t0).Microseconds(),
	// )

	return out, nil
}

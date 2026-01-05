package configmap

import (
	"context"
	"fmt"

	"AtlHyper/atlhyper_agent/utils"
	modelcm "AtlHyper/model/k8s"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ListConfigMaps —— ★ 唯一对外入口：全集群总查询
// 编排：一次性拉全量 corev1.ConfigMap → 为每个 CM 构建“轻量模型”（带预览与大小统计）
func ListConfigMaps(ctx context.Context) ([]modelcm.ConfigMap, error) {
	// t0 := time.Now()
	cs := utils.GetCoreClient()

	// 1) 全集群 ConfigMap 列表
	list, err := cs.CoreV1().ConfigMaps(metav1.NamespaceAll).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list configmaps (cluster-wide) failed: %w", err)
	}
	// t1 := time.Now()

	n := len(list.Items)
	out := make([]modelcm.ConfigMap, 0, n)

	// 2) 构建模型
	var totalKeys, totalBinKeys int
	for i := range list.Items {
		cm := &list.Items[i]
		m := buildModel(cm)
		totalKeys += m.Summary.Keys
		totalBinKeys += m.Summary.BinaryKeys
		out = append(out, m)
	}
	// t2 := time.Now()

	// 3) 日志：阶段耗时（微秒）+ 一些聚合指标
	// log.Printf("[configmap_list] count=%d phases_us list=%d build=%d keys=%d binKeys=%d total=%d",
	// 	n,
	// 	t1.Sub(t0).Microseconds(),
	// 	t2.Sub(t1).Microseconds(),
	// 	totalKeys,
	// 	totalBinKeys,
	// 	t2.Sub(t0).Microseconds(),
	// )

	return out, nil
}

// 避免未使用导入的静态检查（有的 linter 会挑）
var _ corev1.ConfigMap

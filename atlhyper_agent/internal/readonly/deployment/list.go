// internal/readonly/deployment/list.go
package deployment

import (
	"context"
	"fmt"

	"AtlHyper/atlhyper_agent/utils"
	modeldep "AtlHyper/model/deployment"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//
// ListDeployments —— ★ 唯一对外入口：全集群 Deployment 列表
// 编排：一次性拉全量 Deployments → 拉全量 ReplicaSets 建 ownerUID 索引 → 逐个转换为 model
//
func ListDeployments(ctx context.Context) ([]modeldep.Deployment, error) {
	// t0 := time.Now()

	cs := utils.GetCoreClient()

	// 1) 全集群 Deployments
	depList, err := cs.AppsV1().Deployments(metav1.NamespaceAll).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list deployments (cluster-wide) failed: %w", err)
	}
	// t1 := time.Now()

	// 2) 一次性拉全量 ReplicaSets，建立 ownerUID -> []ReplicaSet 索引
	rsIdx, _, err := buildReplicaSetIndex(ctx, cs)
	if err != nil {
		// 索引失败不致命：继续仅以 Deployment 自身信息返回
		rsIdx = nil
	}
	// t2 := time.Now()

	// 3) 转换为模型
	out := make([]modeldep.Deployment, 0, len(depList.Items))
	for i := range depList.Items {
		d := &depList.Items[i]
		out = append(out, buildModel(d, rsIdx))
	}
	// t3 := time.Now()

	// 4) 分段耗时 & 规模日志（微秒）
	// log.Printf("[deployment_list] count=%d rs=%d phases_us list=%d indexRS=%d build=%d total=%d",
	// 	len(depList.Items), totalRS,
	// 	t1.Sub(t0).Microseconds(),
	// 	t2.Sub(t1).Microseconds(),
	// 	t3.Sub(t2).Microseconds(),
	// 	t3.Sub(t0).Microseconds(),
	// )

	return out, nil
}

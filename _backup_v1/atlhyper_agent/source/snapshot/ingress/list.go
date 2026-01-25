// internal/readonly/ingress/list.go
package ingress

import (
	"context"
	"fmt"

	"AtlHyper/atlhyper_agent/sdk"
	modelingr "AtlHyper/model/k8s"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ListIngresses —— ★ 唯一对外入口：全集群 Ingress 查询
// 编排：一次性拉全量 Ingress → 拉 IngressClass（拿 controller）→ 构建模型
func ListIngresses(ctx context.Context) ([]modelingr.Ingress, error) {
	// t0 := time.Now()

	cs := sdk.Get().CoreClient() // client-go *kubernetes.Clientset

	// 1) 全集群 Ingress
	il, err := cs.NetworkingV1().Ingresses(metav1.NamespaceAll).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list ingresses (cluster-wide) failed: %w", err)
	}
	// t1 := time.Now()

	// 2) 可选：拉 IngressClass（取 controller）
	classCtl, _ := fetchIngressClassControllers(ctx) // 不影响主流程
	// t2 := time.Now()

	// 3) 转换
	out := make([]modelingr.Ingress, 0, len(il.Items))
	for i := range il.Items {
		out = append(out, buildModel(&il.Items[i], classCtl))
	}
	// t3 := time.Now()

	// log.Printf("[ingress_list] count=%d phases_us list=%d class=%d build=%d total=%d",
	// 	len(out),
	// 	t1.Sub(t0).Microseconds(),
	// 	t2.Sub(t1).Microseconds(),
	// 	t3.Sub(t2).Microseconds(),
	// 	t3.Sub(t0).Microseconds(),
	// )
	return out, nil
}

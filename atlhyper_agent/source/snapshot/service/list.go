package service

import (
	"context"
	"fmt"

	"AtlHyper/atlhyper_agent/sdk"
	modelsvc "AtlHyper/model/k8s"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ListServices —— ★ 唯一对外入口：全集群总查询
// 编排：一次性拉全量 Service → 构建“静态骨架” → 一次性拉 EndpointSlice（失败则 Endpoints 兜底）→ 就地填充 Backends
func ListServices(ctx context.Context) ([]modelsvc.Service, error) {
	cs := sdk.Get().CoreClient()

	// 1) 全集群 Service 列表
	svcList, err := cs.CoreV1().Services(metav1.NamespaceAll).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list services (cluster-wide) failed: %w", err)
	}

	n := len(svcList.Items)
	out := make([]modelsvc.Service, n)
	keys := make([]string, n) // ns/name 对齐 backends

	// 2) 先构建“静态骨架”（summary/spec/ports/selector/network）
	for i := range svcList.Items {
		svc := &svcList.Items[i]
		out[i] = buildSkeleton(svc)
		keys[i] = svc.Namespace + "/" + svc.Name
	}

	// 3) 尝试 EndpointSlice（优先）
	bidx, slicesCount, err := buildBackendIndexFromSlices(ctx, cs)

	// 若 EndpointSlice 不可用/无结果，则回退到 Endpoints
	if err != nil || slicesCount == 0 {
		var epsCount int
		bidx, epsCount, err = buildBackendIndexFromEndpoints(ctx, cs)
		_ = epsCount // 不再使用
	}

	// 4) 就地填充 backends
	if err == nil && len(bidx) > 0 {
		for i, k := range keys {
			if be, ok := bidx[k]; ok {
				attachBackends(&out[i], be)
			}
		}
	}

	return out, nil
}

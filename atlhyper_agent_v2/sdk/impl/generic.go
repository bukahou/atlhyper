// Package impl K8sClient 接口的具体实现
//
// generic.go - 通用操作
//
// 本文件实现通用的资源操作：
//   - Delete: 通用删除
//   - Dynamic: 动态 API 查询 (仅 GET)
package impl

import (
	"context"
	"fmt"

	"AtlHyper/atlhyper_agent_v2/sdk"
)

// =============================================================================
// 通用操作
// =============================================================================

// Delete 删除资源
//
// TODO: 使用 dynamic client 实现通用删除
func (c *Client) Delete(ctx context.Context, gvk sdk.GroupVersionKind, namespace, name string, opts sdk.DeleteOptions) error {
	return fmt.Errorf("not implemented")
}

// Dynamic 执行动态 API 查询 (仅 GET)
//
// TODO: 使用 rest client 实现 GET 请求
func (c *Client) Dynamic(ctx context.Context, req sdk.DynamicRequest) (*sdk.DynamicResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

// Package impl K8sClient 接口的具体实现
//
// generic.go - 通用操作
//
// 本文件实现通用的资源操作：
//   - Delete: 通用删除
//   - Dynamic: 动态 API 查询 (仅 GET，AI 专用)
package impl

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

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
// 安全限制:
//   - 仅支持 GET 请求 (只读)
//   - 路径必须以 /api/ 或 /apis/ 开头 (合法 K8s API 路径)
//   - 不接受任何请求体
//
// 通过 K8s API Server REST 接口执行只读查询。
// 使用初始化时创建的 httpClient (已配置 TLS 和认证)。
func (c *Client) Dynamic(ctx context.Context, req sdk.DynamicRequest) (*sdk.DynamicResponse, error) {
	if req.Path == "" {
		return nil, fmt.Errorf("path is required")
	}

	// 安全校验: 仅允许合法的 K8s API 路径
	if !strings.HasPrefix(req.Path, "/api/") && !strings.HasPrefix(req.Path, "/apis/") {
		return nil, fmt.Errorf("invalid path: must start with /api/ or /apis/")
	}

	// 构建完整 URL
	u, err := url.Parse(c.config.Host)
	if err != nil {
		return nil, fmt.Errorf("parse host: %w", err)
	}
	u.Path = req.Path
	q := u.Query()
	for k, v := range req.Query {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()

	// 创建 GET 请求
	httpReq, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Accept", "application/json")

	// 执行请求
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	const maxResponseSize = 2 * 1024 * 1024 // 2MB
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	return &sdk.DynamicResponse{
		StatusCode: resp.StatusCode,
		Body:       body,
	}, nil
}

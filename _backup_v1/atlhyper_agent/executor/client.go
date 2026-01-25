// executor/client.go
// 控制循环 HTTP 客户端
package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"AtlHyper/atlhyper_agent/config"
	"AtlHyper/atlhyper_agent/gateway"
)

// Client 控制循环客户端
type Client struct {
	clusterID   string
	waitSeconds int
	watchClient *gateway.HTTPClient
	ackClient   *gateway.HTTPClient
	watchPath   string
	ackPath     string
}

// NewClient 创建控制循环客户端
func NewClient(opsBasePath, clusterID string, waitSeconds int) *Client {
	if waitSeconds <= 0 {
		waitSeconds = 30
	}

	// 使用 RestClient.BaseURL（与 pusher 保持一致）
	baseURL := config.GlobalConfig.RestClient.BaseURL

	// Watch 客户端：超时要大于 waitSeconds
	watchClient := gateway.NewHTTPClient(gateway.ClientConfig{
		BaseURL: baseURL,
		Timeout: time.Duration(waitSeconds+10) * time.Second,
	})

	// Ack 客户端：正常很快
	ackClient := gateway.NewHTTPClient(gateway.ClientConfig{
		BaseURL: baseURL,
		Timeout: 8 * time.Second,
	})

	return &Client{
		clusterID:   clusterID,
		waitSeconds: waitSeconds,
		watchClient: watchClient,
		ackClient:   ackClient,
		watchPath:   opsBasePath + "/watch",
		ackPath:     opsBasePath + "/ack",
	}
}

// Watch 长轮询监听命令
// 返回 (*CommandSet, changed, error)，changed=false 表示 304 无更新
func (c *Client) Watch(ctx context.Context, rv uint64) (*CommandSet, bool, error) {
	req := map[string]any{
		"clusterID":   c.clusterID,
		"rv":          fmt.Sprintf("%d", rv),
		"waitSeconds": c.waitSeconds,
	}

	code, body, err := c.watchClient.Post(ctx, c.watchPath, req)
	if err != nil {
		return nil, false, err
	}
	if code == http.StatusNotModified {
		return nil, false, nil
	}
	if code != http.StatusOK {
		return nil, false, fmt.Errorf("watch unexpected status: %d", code)
	}

	var set CommandSet
	if err := json.Unmarshal(body, &set); err != nil {
		return nil, false, err
	}
	return &set, true, nil
}

// Ack 批量回执执行结果
func (c *Client) Ack(ctx context.Context, results []AckResult) error {
	req := map[string]any{
		"clusterID": c.clusterID,
		"results":   results,
	}

	code, _, err := c.ackClient.Post(ctx, c.ackPath, req)
	if err != nil {
		return err
	}
	if code != http.StatusOK {
		return fmt.Errorf("ack unexpected status: %d", code)
	}
	return nil
}

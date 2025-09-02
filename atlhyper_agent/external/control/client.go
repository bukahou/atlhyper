package control

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"AtlHyper/atlhyper_agent/external/push/client"
	pcfg "AtlHyper/atlhyper_agent/external/push/config"
)

// Client 基于 push 的 Sender 抽象，按 path 驱动
type Client struct {
	clusterID   string
	waitSeconds int
	watchSender client.Sender // POST {clusterID, rv, waitSeconds} -> 200: CommandSet / 304: no change
	ackSender   client.Sender // POST {clusterID, results}
}

// external/control/client.go
func NewClient(opsBasePath, clusterID string, waitSeconds int) *Client {
    if waitSeconds <= 0 { waitSeconds = 30 }

    wcfg := pcfg.NewDefaultRestClientConfig()
    wcfg.Path = opsBasePath + "/watch"
    // ★ 关键：超时要大于 waitSeconds
    wcfg.Timeout = time.Duration(waitSeconds+10) * time.Second

    acfg := pcfg.NewDefaultRestClientConfig()
    acfg.Path = opsBasePath + "/ack"
    // ack 正常很快，不需要很大
    acfg.Timeout = 8 * time.Second

    return &Client{
        clusterID:   clusterID,
        waitSeconds: waitSeconds,
        watchSender: client.NewSender(wcfg),
        ackSender:   client.NewSender(acfg),
    }
}


// Watch：返回 (*CommandSet, changed, error)
// changed=false → 304 无更新
func (c *Client) Watch(ctx context.Context, rv uint64) (*CommandSet, bool, error) {
	req := map[string]any{
		"clusterID":   c.clusterID,
		"rv":          fmt.Sprintf("%d", rv),
		"waitSeconds": c.waitSeconds,
	}
	code, body, err := c.watchSender.Post(ctx, req)
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

// Ack：批量回执执行结果
func (c *Client) Ack(ctx context.Context, results []AckResult) error {
	req := map[string]any{
		"clusterID": c.clusterID,
		"results":   results,
	}
	code, _, err := c.ackSender.Post(ctx, req)
	if err != nil {
		return err
	}
	if code != http.StatusOK {
		return fmt.Errorf("ack unexpected status: %d", code)
	}
	return nil
}

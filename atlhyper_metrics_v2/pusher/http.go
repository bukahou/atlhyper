// Package pusher 指标推送
package pusher

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"AtlHyper/atlhyper_metrics_v2/config"
	"AtlHyper/model_v2"
)

// HTTPPusher HTTP 推送器
type HTTPPusher struct {
	cfg    *config.Config
	client *http.Client
}

// New 创建 HTTP 推送器
func New(cfg *config.Config) *HTTPPusher {
	return &HTTPPusher{
		cfg: cfg,
		client: &http.Client{
			Timeout: cfg.Push.Timeout,
		},
	}
}

// Push 推送指标到 Agent
func (p *HTTPPusher) Push(snapshot *model_v2.NodeMetricsSnapshot) error {
	url := p.cfg.Push.AgentAddr + "/metrics/node"

	// 序列化
	data, err := json.Marshal(snapshot)
	if err != nil {
		return fmt.Errorf("marshal snapshot: %w", err)
	}

	// 重试逻辑
	var lastErr error
	for i := 0; i < p.cfg.Push.RetryCount; i++ {
		if i > 0 {
			time.Sleep(p.cfg.Push.RetryDelay)
			log.Printf("[Pusher] 重试推送 (%d/%d)", i+1, p.cfg.Push.RetryCount)
		}

		err = p.doRequest(url, data)
		if err == nil {
			return nil
		}
		lastErr = err
	}

	return fmt.Errorf("push failed after %d retries: %w", p.cfg.Push.RetryCount, lastErr)
}

// doRequest 执行 HTTP 请求
func (p *HTTPPusher) doRequest(url string, data []byte) error {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

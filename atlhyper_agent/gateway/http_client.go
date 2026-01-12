// gateway/http_client.go
// HTTP 客户端封装 (推送/长轮询)
package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	ziputil "AtlHyper/common"
)

// ClientConfig HTTP 客户端配置
type ClientConfig struct {
	BaseURL      string
	Timeout      time.Duration
	MaxRespBytes int64
	EnableGzip   bool
	Headers      map[string]string
}

// HTTPClient HTTP 客户端
type HTTPClient struct {
	cfg    ClientConfig
	client *http.Client
}

// NewHTTPClient 创建 HTTP 客户端
func NewHTTPClient(cfg ClientConfig) *HTTPClient {
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30 * time.Second
	}
	if cfg.MaxRespBytes <= 0 {
		cfg.MaxRespBytes = 10 * 1024 * 1024 // 10MB
	}

	transport := &http.Transport{
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	return &HTTPClient{
		cfg: cfg,
		client: &http.Client{
			Timeout:   cfg.Timeout,
			Transport: transport,
		},
	}
}

// Post 发送 POST 请求
func (c *HTTPClient) Post(ctx context.Context, path string, payload any) (int, []byte, error) {
	// 序列化
	body, err := json.Marshal(payload)
	if err != nil {
		return 0, nil, err
	}

	var reader io.Reader = bytes.NewReader(body)
	contentEncoding := ""

	// 可选 gzip 压缩
	if c.cfg.EnableGzip {
		gz, err := ziputil.GzipBytes(body)
		if err != nil {
			return 0, nil, err
		}
		reader = bytes.NewReader(gz)
		contentEncoding = "gzip"
	}

	// 构造请求
	url := c.cfg.BaseURL + path
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, reader)
	if err != nil {
		return 0, nil, err
	}

	// 设置头部
	req.Header.Set("Content-Type", "application/json")
	if contentEncoding != "" {
		req.Header.Set("Content-Encoding", contentEncoding)
	}
	for k, v := range c.cfg.Headers {
		req.Header.Set(k, v)
	}

	// 发送
	resp, err := c.client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	// 读取响应
	limited := io.LimitReader(resp.Body, c.cfg.MaxRespBytes)
	respBody, err := io.ReadAll(limited)
	if err != nil {
		return resp.StatusCode, nil, err
	}

	return resp.StatusCode, respBody, nil
}

// Get 发送 GET 请求
func (c *HTTPClient) Get(ctx context.Context, path string) (int, []byte, error) {
	url := c.cfg.BaseURL + path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, nil, err
	}

	for k, v := range c.cfg.Headers {
		req.Header.Set(k, v)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	limited := io.LimitReader(resp.Body, c.cfg.MaxRespBytes)
	respBody, err := io.ReadAll(limited)
	if err != nil {
		return resp.StatusCode, nil, err
	}

	return resp.StatusCode, respBody, nil
}

// Watch 长轮询请求
func (c *HTTPClient) Watch(ctx context.Context, path string, timeout time.Duration) (int, []byte, error) {
	url := c.cfg.BaseURL + path

	// 创建带超时的 context
	watchCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(watchCtx, http.MethodGet, url, nil)
	if err != nil {
		return 0, nil, err
	}

	for k, v := range c.cfg.Headers {
		req.Header.Set(k, v)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	limited := io.LimitReader(resp.Body, c.cfg.MaxRespBytes)
	respBody, err := io.ReadAll(limited)
	if err != nil {
		return resp.StatusCode, nil, err
	}

	return resp.StatusCode, respBody, nil
}

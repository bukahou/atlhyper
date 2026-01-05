package client

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	restcfg "AtlHyper/atlhyper_agent/external/push/config"
	ziputil "AtlHyper/common" // ★ 公共 gzip 工具（根目录 utils）
)

type RestfulClient struct {
	baseURL        string
	path           string
	timeout        time.Duration
	maxRespBytes   int64
	defaultHeaders map[string]string
	client         *http.Client
}

func NewRestfulClient(cfg restcfg.RestClientConfig) *RestfulClient {
	transport := &http.Transport{
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	return &RestfulClient{
		baseURL:        cfg.BaseURL,
		path:           cfg.Path,
		timeout:        cfg.Timeout,
		maxRespBytes:   cfg.MaxRespBytes,
		defaultHeaders: cfg.DefaultHeaders,
		client: &http.Client{
			Timeout:   cfg.Timeout,
			Transport: transport,
		},
	}
}

// Post 实现 Sender 接口：使用配置内的 path，外部无需关心 URL 细节。
func (c *RestfulClient) Post(ctx context.Context, payload any) (int, []byte, error) {
	// 1) 序列化为 JSON
	body, err := json.Marshal(payload)
	if err != nil {
		return 0, nil, err
	}

	// 2) gzip 压缩整包（统一走压缩；Master 端会自动解压/透传）
	gz, err := ziputil.GzipBytes(body)
	if err != nil {
		return 0, nil, err
	}
	reader := bytes.NewReader(gz)

	// 3) 构造请求
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+c.path, reader)
	if err != nil {
		return 0, nil, err
	}

	// 4) 头部（声明压缩）
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip") // ★ 关键：声明请求体为 gzip
	// 不显式设置 Accept-Encoding，保持 Go http 自动解压响应的默认行为

	for k, v := range c.defaultHeaders {
		// 保护基本头，避免被覆盖
		if k == "Content-Type" || k == "Content-Encoding" {
			continue
		}
		req.Header.Set(k, v)
	}

	// 5) 发送 + 限流读取响应
	resp, err := c.client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	limited := io.LimitReader(resp.Body, c.maxRespBytes)
	respBody, rerr := io.ReadAll(limited)
	if rerr != nil {
		return resp.StatusCode, nil, rerr
	}
	return resp.StatusCode, respBody, nil
}

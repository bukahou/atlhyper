// logic/pusher/generic.go
// 通用推送器实现 (消除重复代码)
package pusher

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"reflect"
	"time"

	ziputil "AtlHyper/common"
)

// Config 推送器配置
type Config struct {
	Name       string        // 推送器名称
	ClusterID  string        // 集群ID
	Source     string        // Envelope Source 标识
	Path       string        // 推送路径
	BaseURL    string        // Master 地址
	Interval   time.Duration // 推送间隔
	Timeout    time.Duration // HTTP 超时
	MaxRetries int           // 最大重试次数
}

// GenericPusher 通用推送器
type GenericPusher struct {
	cfg        Config
	dataSource DataSource
	httpClient *http.Client
	stopCh     chan struct{}
}

// NewGenericPusher 创建通用推送器
func NewGenericPusher(cfg Config, dataSource DataSource) *GenericPusher {
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30 * time.Second
	}
	if cfg.MaxRetries <= 0 {
		cfg.MaxRetries = 3
	}

	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     90 * time.Second,
	}

	return &GenericPusher{
		cfg:        cfg,
		dataSource: dataSource,
		httpClient: &http.Client{
			Timeout:   cfg.Timeout,
			Transport: transport,
		},
		stopCh: make(chan struct{}),
	}
}

// Name 推送器名称
func (p *GenericPusher) Name() string {
	return p.cfg.Name
}

// Push 执行一次推送
func (p *GenericPusher) Push(ctx context.Context) error {
	// 1. 获取数据
	data, err := p.dataSource.Fetch(ctx)
	if err != nil {
		log.Printf("[%s_pusher] fetch error: %v", p.cfg.Name, err)
		return fmt.Errorf("fetch data: %w", err)
	}

	// 2. 检查是否为空
	if isEmpty(data) {
		return nil
	}

	// 3. 序列化
	payload, err := json.Marshal(data)
	if err != nil {
		log.Printf("[%s_pusher] marshal error: %v", p.cfg.Name, err)
		return fmt.Errorf("marshal: %w", err)
	}

	// 4. 构造 Envelope
	env := NewEnvelope(p.cfg.ClusterID, p.cfg.Source, payload)

	// 5. 推送 (带重试)
	err = p.pushWithRetry(ctx, env)
	if err != nil {
		log.Printf("[%s_pusher] push failed: %v", p.cfg.Name, err)
	}
	return err
}

// pushWithRetry 带重试的推送
func (p *GenericPusher) pushWithRetry(ctx context.Context, env any) error {
	var lastErr error

	for i := 0; i < p.cfg.MaxRetries; i++ {
		if i > 0 {
			time.Sleep(time.Duration(i) * 100 * time.Millisecond)
		}

		code, err := p.doPost(ctx, env)
		if err == nil && code >= 200 && code < 300 {
			return nil
		}

		lastErr = fmt.Errorf("http=%d err=%v", code, err)
	}

	return lastErr
}

// doPost 执行 HTTP POST
func (p *GenericPusher) doPost(ctx context.Context, payload any) (int, error) {
	// 序列化
	body, err := json.Marshal(payload)
	if err != nil {
		return 0, err
	}

	// Gzip 压缩
	gz, err := ziputil.GzipBytes(body)
	if err != nil {
		return 0, err
	}

	// 构造请求
	url := p.cfg.BaseURL + p.cfg.Path
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(gz))
	if err != nil {
		return 0, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")

	// 发送
	resp, err := p.httpClient.Do(req)
	if err != nil {
		log.Printf("[%s_pusher] http error: url=%s err=%v", p.cfg.Name, url, err)
		return 0, err
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))

	if resp.StatusCode >= 400 {
		log.Printf("[%s_pusher] http %d: url=%s resp=%s", p.cfg.Name, resp.StatusCode, url, string(respBody))
	}

	return resp.StatusCode, nil
}

// Start 启动定时推送
func (p *GenericPusher) Start(ctx context.Context) {
	// 立即推送一次
	if err := p.Push(ctx); err != nil {
		log.Printf("[%s_pusher] initial push error: %v", p.cfg.Name, err)
	}

	go func() {
		ticker := time.NewTicker(p.cfg.Interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-p.stopCh:
				return
			case <-ticker.C:
				if err := p.Push(ctx); err != nil {
					log.Printf("[%s_pusher] error: %v", p.cfg.Name, err)
				}
			}
		}
	}()

	log.Printf("[%s_pusher] started, interval=%v", p.cfg.Name, p.cfg.Interval)
}

// Stop 停止推送
func (p *GenericPusher) Stop() {
	close(p.stopCh)
}

// isEmpty 检查数据是否为空
func isEmpty(data any) bool {
	if data == nil {
		return true
	}

	v := reflect.ValueOf(data)
	switch v.Kind() {
	case reflect.Slice, reflect.Map, reflect.Array:
		return v.Len() == 0
	case reflect.Ptr, reflect.Interface:
		if v.IsNil() {
			return true
		}
		return isEmpty(v.Elem().Interface())
	}

	return false
}

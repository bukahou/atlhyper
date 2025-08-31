package push

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"AtlHyper/atlhyper_metrics/config"
	"AtlHyper/atlhyper_metrics/internal"
)

// ==========================
// 📦 模块说明
// ==========================
// 本模块负责将 metrics 采集端（Collector）
// 定期聚合的节点指标快照（NodeMetricsSnapshot）
// 主动推送 (Push) 给 Agent 的接收端。
//
// ✅ 关键特性：
// - 通过 config.C 读取的环境变量配置控制是否启用、推送间隔、目标地址等
// - 启动时立即推送一次，避免空窗
// - 数据使用 gzip 压缩后以 JSON 发送，减少网络带宽
// - 带有限重试与指数退避
//
// ==========================
// 🌍 配置来源
// ==========================
// 请在进程启动时调用 config.MustLoad()，该函数会从环境变量读取：
// PUSH_ENABLE, PUSH_URL, PUSH_TOKEN, PUSH_INTERVAL, PUSH_TIMEOUT
// 并填充到 config.C.Push 中。

// pusher 封装了 Push 上报的配置与 HTTP 客户端
type pusher struct {
	url      string        // 接收端 URL
	token    string        // 可选的 Bearer Token
	client   *http.Client  // HTTP 客户端（带超时）
	interval time.Duration // 上报间隔
}

// StartReporterFromEnv 使用已加载到 config.C 的配置；名称保持不变以兼容调用方。
// 如果启用（config.C.Push.Enable=true），则启动后台上报任务。
func StartReporterFromEnv(ctx context.Context) {
	cfg := config.C.Push

	// 未启用则直接返回（避免启动 goroutine）
    if !cfg.Enable {
        log.Println("ℹ️ [Push] disabled (PUSH_ENABLE=false or unset)")
        return
    }

	// 启用但缺少 URL，跳过上报
	if cfg.URL == "" {
		log.Println("⚠️ [Push] PUSH_ENABLE=true 但未配置 PUSH_URL，跳过上报")
		return
	}

	// 使用已在 config.Load() 中解析好的间隔与超时（都有默认值）
	p := &pusher{
		url:      cfg.URL,
		token:    cfg.Token, // 允许为空：为空则不携带 Authorization 头
		client:   &http.Client{Timeout: cfg.Timeout},
		interval: cfg.Interval,
	}

	// 后台循环上报
	go p.loop(ctx)

	log.Printf("📤 [Push] 上报已启用：url=%s interval=%s timeout=%s", cfg.URL, cfg.Interval, cfg.Timeout)
}

// loop 按固定间隔循环上报
func (p *pusher) loop(ctx context.Context) {
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	// 启动后立即推送一次，避免等待一个完整间隔
	p.pushOnce(ctx)

	for {
		select {
		case <-ctx.Done():
			// 收到取消信号，退出循环
			return
		case <-ticker.C:
			// 到达间隔时间，执行一次推送
			p.pushOnce(ctx)
		}
	}
}

// pushOnce 执行一次快照采集与推送
func (p *pusher) pushOnce(ctx context.Context) {
	// 从采集模块聚合当前节点快照（CPU 已从缓存读取）
	snap := internal.BuildNodeMetricsSnapshot()
	if snap == nil {
		return
	}

	// 序列化为 JSON
	payload, err := json.Marshal(snap)
	if err != nil {
		log.Printf("❌ [Push] 序列化失败: %v", err)
		return
	}

	// 使用 gzip 压缩以减少传输体积
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	if _, err := zw.Write(payload); err != nil {
		log.Printf("❌ [Push] gzip 失败: %v", err)
		_ = zw.Close()
		return
	}
	_ = zw.Close()

	// 最多重试 3 次，采用指数退避：250ms, 500ms, 1s
	const maxRetry = 3
	var lastErr error
	for i := 0; i < maxRetry; i++ {
		if err := doPostGzipJSON(ctx, p.client, p.url, buf.Bytes(), p.token); err == nil {
			// 推送成功
			return
		} else {
			lastErr = err
			backoff := time.Duration(250*(1<<i)) * time.Millisecond
			time.Sleep(backoff)
		}
	}

	// 所有重试失败，打印错误日志
	log.Printf("❌ [Push] 上报失败（重试 %d 次）: %v", maxRetry, lastErr)
}

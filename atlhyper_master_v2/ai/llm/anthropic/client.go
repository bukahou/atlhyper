// atlhyper_master_v2/ai/llm/anthropic/client.go
// Anthropic Claude LLM 客户端（官方 Go SDK，流式 Messages 接口）
package anthropic

import (
	"context"
	"sync"

	"AtlHyper/atlhyper_master_v2/ai/llm"
	"AtlHyper/common/logger"

	anthropicsdk "github.com/anthropics/anthropic-sdk-go/v2"
	"github.com/anthropics/anthropic-sdk-go/v2/option"
)

var log = logger.Module("Anthropic")

const (
	defaultModel    = "claude-haiku-5-5"
	maxOutputTokens = 32000 // 思考也计入输出上限
)

func init() {
	llm.Register("anthropic", func(cfg llm.Config) (llm.LLMClient, error) {
		return NewAnthropicClient(cfg)
	})
}

// Client Claude 客户端：AI 服务每次请求建一个，底层 SDK 客户端按认证方式复用（见 NewAnthropicClient）
type Client struct {
	api   *anthropicsdk.Client
	model string
}

// NewAnthropicClient 创建客户端（不发请求）。
//
//   - 配了 key：用这把 key 新建 SDK 客户端。
//   - 没配 key：交给 SDK 的凭证链。集群里是 Workload Identity Federation：环境变量 ANTHROPIC_FEDERATION_RULE_ID /
//     ANTHROPIC_ORGANIZATION_ID / ANTHROPIC_SERVICE_ACCOUNT_ID / ANTHROPIC_IDENTITY_TOKEN_FILE + 投射的服务账号令牌，
//     SDK 换成短期访问令牌并自动续期。⚠️ 进程环境里的 ANTHROPIC_API_KEY 在凭证链里排第一，会盖过 WIF。
//
// ⛔ 没配 key 的路径全进程共用一个 SDK 客户端（按地址各一个）：SDK 按客户端各存一份令牌，集群令牌带 jti、
// 同一张只能换一次，而 AI 服务每次请求都会新建 LLMClient —— 每次新建 SDK 客户端的话，第二次起换令牌就会 401。
func NewAnthropicClient(cfg llm.Config) (*Client, error) {
	model := cfg.Model
	if model == "" {
		model = defaultModel
	}
	if cfg.APIKey == "" {
		return &Client{api: keylessClient(cfg.BaseURL), model: model}, nil
	}
	options := []option.RequestOption{option.WithAPIKey(cfg.APIKey)}
	if cfg.BaseURL != "" {
		options = append(options, option.WithBaseURL(cfg.BaseURL))
	}
	api := anthropicsdk.NewClient(options...)
	return &Client{api: &api, model: model}, nil
}

var (
	keylessMu      sync.Mutex
	keylessClients = map[string]*anthropicsdk.Client{} // 地址 → 免密钥客户端
)

// keylessClient 取该地址的免密钥客户端，第一次用时创建
func keylessClient(baseURL string) *anthropicsdk.Client {
	keylessMu.Lock()
	defer keylessMu.Unlock()
	if api, ok := keylessClients[baseURL]; ok {
		return api
	}
	var options []option.RequestOption
	if baseURL != "" {
		options = append(options, option.WithBaseURL(baseURL))
	}
	api := anthropicsdk.NewClient(options...)
	keylessClients[baseURL] = &api
	return &api
}

// ChatStream 发送流式对话请求。文字增量即时发出；工具调用、用量与整轮回答在流结束后发出。
func (c *Client) ChatStream(ctx context.Context, req *llm.Request) (<-chan *llm.Chunk, error) {
	params, err := buildParams(c.model, req)
	if err != nil {
		return nil, err
	}
	log.Debug("请求", "model", c.model, "messages", len(params.Messages), "tools", len(params.Tools))

	stream := c.api.Messages.NewStreaming(ctx, params)
	ch := make(chan *llm.Chunk, 32)
	go func() {
		defer close(ch)
		defer stream.Close()
		pumpStream(ctx, stream, ch)
	}()
	return ch, nil
}

// Close 无需释放资源（HTTP 连接由 SDK 管理；免密钥客户端全进程共用）
func (c *Client) Close() error { return nil }

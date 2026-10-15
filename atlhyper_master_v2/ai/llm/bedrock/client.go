// atlhyper_master_v2/ai/llm/bedrock/client.go
// AWS Bedrock LLM 客户端（ConverseStream，API key 以 Bearer 鉴权）
package bedrock

import (
	"context"
	"fmt"

	"AtlHyper/atlhyper_master_v2/ai/llm"

	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/smithy-go/auth/bearer"
)

func init() {
	llm.Register("bedrock", func(cfg llm.Config) (llm.LLMClient, error) {
		return NewBedrockClient(cfg)
	})
}

// Client Bedrock 客户端
type Client struct {
	api   *bedrockruntime.Client
	model string
}

// NewBedrockClient 创建 Bedrock 客户端
//
// 鉴权只用传入的 API key（Bearer）：
//   - 不调用 config.LoadDefaultConfig → 不走 AWS 凭证链，SigV4 解析器为 nil，自然落到 Bearer
//   - ⚠️ key 必须在 optFn 里设置：SDK 的 New() 在执行 optFn 之前会读环境变量
//     AWS_BEARER_TOKEN_BEDROCK 并覆盖 Options 字面量中的 token；写在 optFn 里则以我方为准
//
// extra 供测试替换端点 / HTTP 客户端；鉴权设置总在最后执行，不会被覆盖。
func NewBedrockClient(cfg llm.Config, extra ...func(*bedrockruntime.Options)) (*Client, error) {
	switch {
	case cfg.APIKey == "":
		return nil, fmt.Errorf("bedrock api key is required")
	case cfg.Model == "":
		return nil, fmt.Errorf("bedrock model is required")
	case cfg.Region == "":
		return nil, fmt.Errorf("bedrock region is required（Master 配置 MASTER_AI_BEDROCK_REGION）")
	}

	apiKey := cfg.APIKey
	optFns := append(extra, func(o *bedrockruntime.Options) {
		o.BearerAuthTokenProvider = bearer.StaticTokenProvider{Token: bearer.Token{Value: apiKey}}
		o.AuthSchemePreference = []string{"httpBearerAuth"}
	})
	api := bedrockruntime.New(bedrockruntime.Options{Region: cfg.Region}, optFns...)
	return &Client{api: api, model: cfg.Model}, nil
}

// ChatStream 发送流式对话请求
func (c *Client) ChatStream(ctx context.Context, req *llm.Request) (<-chan *llm.Chunk, error) {
	input, err := buildInput(c.model, req)
	if err != nil {
		return nil, err
	}
	resp, err := c.api.ConverseStream(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("bedrock ConverseStream: %w", err)
	}

	stream := resp.GetStream()
	ch := make(chan *llm.Chunk, 16)
	go func() {
		defer close(ch)
		defer stream.Close()
		pump(ctx, stream.Events(), stream.Err, ch)
	}()
	return ch, nil
}

// Close 无需释放资源（HTTP 连接由 SDK 管理）
func (c *Client) Close() error { return nil }

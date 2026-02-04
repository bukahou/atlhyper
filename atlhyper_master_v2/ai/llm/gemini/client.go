// atlhyper_master_v2/ai/llm/gemini/client.go
// Gemini LLM 客户端实现
package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"

	"AtlHyper/atlhyper_master_v2/ai/llm"
)

func init() {
	llm.Register("gemini", func(apiKey, model string) (llm.LLMClient, error) {
		return New(apiKey, model)
	})
}

// Client Gemini 客户端
type Client struct {
	client *genai.Client
	model  string
}

// New 创建 Gemini 客户端
func New(apiKey, model string) (*Client, error) {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create gemini client: %w", err)
	}
	return &Client{client: client, model: model}, nil
}

// ChatStream 发送流式对话请求
func (c *Client) ChatStream(ctx context.Context, req *llm.Request) (<-chan *llm.Chunk, error) {
	model := c.client.GenerativeModel(c.model)

	// 设置系统提示词
	if req.SystemPrompt != "" {
		model.SystemInstruction = genai.NewUserContent(genai.Text(req.SystemPrompt))
	}

	// 设置 Tools
	if len(req.Tools) > 0 {
		model.Tools = convertTools(req.Tools)
	}

	// 构建 Chat Session
	cs := model.StartChat()

	// 设置历史消息（除最后一条外）
	history, lastParts := splitMessages(req.Messages)
	cs.History = history

	// 发送最后一条消息
	iter := cs.SendMessageStream(ctx, lastParts...)

	// 启动流式读取
	ch := make(chan *llm.Chunk, 32)
	go func() {
		defer close(ch)
		c.readStream(ctx, iter, ch)
	}()

	return ch, nil
}

// readStream 读取流式响应并转换为 Chunk
func (c *Client) readStream(ctx context.Context, iter *genai.GenerateContentResponseIterator, ch chan<- *llm.Chunk) {
	var lastUsage *llm.Usage
	for {
		resp, err := iter.Next()
		if err == iterator.Done {
			ch <- &llm.Chunk{Type: llm.ChunkDone, Usage: lastUsage}
			return
		}
		if err != nil {
			log.Printf("[Gemini] 流式读取错误: %v", err)
			ch <- &llm.Chunk{Type: llm.ChunkError, Error: err}
			return
		}

		// 记录 usage（每次响应可能包含 usage，取最后一个）
		if resp.UsageMetadata != nil {
			lastUsage = &llm.Usage{
				InputTokens:  int(resp.UsageMetadata.PromptTokenCount),
				OutputTokens: int(resp.UsageMetadata.CandidatesTokenCount),
			}
		}

		// 解析候选响应
		for _, cand := range resp.Candidates {
			if cand.Content == nil {
				continue
			}
			for _, part := range cand.Content.Parts {
				chunk := convertPart(part)
				if chunk != nil {
					select {
					case ch <- chunk:
					case <-ctx.Done():
						ch <- &llm.Chunk{Type: llm.ChunkError, Error: ctx.Err()}
						return
					}
				}
			}
		}
	}
}

// convertPart 将 genai.Part 转换为 Chunk
func convertPart(part genai.Part) *llm.Chunk {
	switch p := part.(type) {
	case genai.Text:
		if string(p) == "" {
			return nil
		}
		return &llm.Chunk{
			Type:    llm.ChunkText,
			Content: string(p),
		}
	case genai.FunctionCall:
		paramsJSON, _ := json.Marshal(p.Args)
		return &llm.Chunk{
			Type: llm.ChunkToolCall,
			ToolCall: &llm.ToolCall{
				ID:     fmt.Sprintf("call_%s", p.Name),
				Name:   p.Name,
				Params: string(paramsJSON),
			},
		}
	default:
		return nil
	}
}

// splitMessages 分割消息为历史和最后一条
// 返回: (历史 Content 列表, 最后一条消息的 Parts)
//
// 关键处理: 合并连续的 function role Content。
// Gemini 要求一轮 N 个 FunctionCall 对应的 FunctionResponse 必须在同一个 Content 中。
func splitMessages(msgs []llm.Message) ([]*genai.Content, []genai.Part) {
	if len(msgs) == 0 {
		return nil, []genai.Part{genai.Text("")}
	}

	// 将所有消息转为 Content
	var rawContents []*genai.Content
	for _, msg := range msgs {
		content := convertMessage(msg)
		if content != nil {
			rawContents = append(rawContents, content)
		}
	}

	if len(rawContents) == 0 {
		return nil, []genai.Part{genai.Text("")}
	}

	// 合并连续的 function role Content
	contents := mergeFunctionContents(rawContents)

	// 分割: 历史 + 最后一条的 Parts
	history := contents[:len(contents)-1]
	lastParts := contents[len(contents)-1].Parts

	return history, lastParts
}

// mergeFunctionContents 合并连续的 role=function Content
// 例: [model, function, function] → [model, function(合并)]
func mergeFunctionContents(contents []*genai.Content) []*genai.Content {
	var merged []*genai.Content
	for _, c := range contents {
		if c.Role == "function" && len(merged) > 0 && merged[len(merged)-1].Role == "function" {
			// 追加 Parts 到前一个 function Content
			merged[len(merged)-1].Parts = append(merged[len(merged)-1].Parts, c.Parts...)
		} else {
			merged = append(merged, c)
		}
	}
	return merged
}

// convertMessage 将 llm.Message 转换为 genai.Content
func convertMessage(msg llm.Message) *genai.Content {
	switch msg.Role {
	case "user":
		return genai.NewUserContent(genai.Text(msg.Content))
	case "assistant":
		parts := []genai.Part{}
		if msg.Content != "" {
			parts = append(parts, genai.Text(msg.Content))
		}
		for _, tc := range msg.ToolCalls {
			var args map[string]any
			json.Unmarshal([]byte(tc.Params), &args)
			parts = append(parts, genai.FunctionCall{
				Name: tc.Name,
				Args: args,
			})
		}
		if len(parts) == 0 {
			return nil
		}
		return &genai.Content{
			Role:  "model",
			Parts: parts,
		}
	case "tool":
		if msg.ToolResult == nil {
			return nil
		}
		var result map[string]any
		if err := json.Unmarshal([]byte(msg.ToolResult.Content), &result); err != nil {
			// 非 JSON 结果包装为 output 字段
			result = map[string]any{"output": msg.ToolResult.Content}
		}
		return &genai.Content{
			Role: "function",
			Parts: []genai.Part{
				genai.FunctionResponse{
					Name:     msg.ToolResult.Name,
					Response: result,
				},
			},
		}
	default:
		return nil
	}
}

// convertTools 将 ToolDefinition 列表转换为 genai.Tool
func convertTools(tools []llm.ToolDefinition) []*genai.Tool {
	var declarations []*genai.FunctionDeclaration
	for _, t := range tools {
		fd := &genai.FunctionDeclaration{
			Name:        t.Name,
			Description: t.Description,
		}
		// 解析 JSON Schema 为 genai.Schema
		if len(t.Parameters) > 0 {
			schema := parseSchema(t.Parameters)
			if schema != nil {
				fd.Parameters = schema
			}
		}
		declarations = append(declarations, fd)
	}
	return []*genai.Tool{
		{FunctionDeclarations: declarations},
	}
}

// parseSchema 将 JSON Schema 转换为 genai.Schema
func parseSchema(raw json.RawMessage) *genai.Schema {
	var schemaMap map[string]any
	if err := json.Unmarshal(raw, &schemaMap); err != nil {
		return nil
	}
	return buildSchema(schemaMap)
}

// buildSchema 递归构建 genai.Schema
func buildSchema(m map[string]any) *genai.Schema {
	schema := &genai.Schema{}

	if t, ok := m["type"].(string); ok {
		switch t {
		case "object":
			schema.Type = genai.TypeObject
		case "string":
			schema.Type = genai.TypeString
		case "number":
			schema.Type = genai.TypeNumber
		case "integer":
			schema.Type = genai.TypeInteger
		case "boolean":
			schema.Type = genai.TypeBoolean
		case "array":
			schema.Type = genai.TypeArray
		}
	}

	if desc, ok := m["description"].(string); ok {
		schema.Description = desc
	}

	// 处理 properties
	if props, ok := m["properties"].(map[string]any); ok {
		schema.Properties = make(map[string]*genai.Schema)
		for key, val := range props {
			if propMap, ok := val.(map[string]any); ok {
				schema.Properties[key] = buildSchema(propMap)
			}
		}
	}

	// 处理 required
	if req, ok := m["required"].([]any); ok {
		for _, r := range req {
			if s, ok := r.(string); ok {
				schema.Required = append(schema.Required, s)
			}
		}
	}

	// 处理 items (array)
	if items, ok := m["items"].(map[string]any); ok {
		schema.Items = buildSchema(items)
	}

	// 处理 enum
	if enum, ok := m["enum"].([]any); ok {
		for _, e := range enum {
			if s, ok := e.(string); ok {
				schema.Enum = append(schema.Enum, s)
			}
		}
	}

	return schema
}

// Close 关闭客户端
func (c *Client) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

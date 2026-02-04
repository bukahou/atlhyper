// atlhyper_master_v2/ai/llm/anthropic/client.go
// Anthropic Claude LLM 客户端实现
package anthropic

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"AtlHyper/atlhyper_master_v2/ai/llm"
)

const (
	apiEndpoint   = "https://api.anthropic.com/v1/messages"
	apiVersion    = "2023-06-01"
	maxTokens     = 4096
)

func init() {
	llm.Register("anthropic", func(apiKey, model string) (llm.LLMClient, error) {
		return New(apiKey, model)
	})
}

// Client Anthropic Claude 客户端
type Client struct {
	apiKey     string
	model      string
	httpClient *http.Client
}

// New 创建 Anthropic 客户端
func New(apiKey, model string) (*Client, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("anthropic api key is required")
	}
	if model == "" {
		model = "claude-sonnet-4-20250514"
	}
	return &Client{
		apiKey:     apiKey,
		model:      model,
		httpClient: &http.Client{},
	}, nil
}

// ==================== API Request/Response Types ====================

// messagesRequest Anthropic Messages API リクエスト
type messagesRequest struct {
	Model       string          `json:"model"`
	MaxTokens   int             `json:"max_tokens"`
	System      string          `json:"system,omitempty"`
	Messages    []messageParam  `json:"messages"`
	Tools       []toolParam     `json:"tools,omitempty"`
	Stream      bool            `json:"stream"`
}

// messageParam メッセージパラメータ
type messageParam struct {
	Role    string        `json:"role"`
	Content []contentPart `json:"content"`
}

// contentPart コンテンツパート（テキスト、ツール呼び出し、ツール結果）
type contentPart struct {
	Type      string `json:"type"`
	Text      string `json:"text,omitempty"`
	ID        string `json:"id,omitempty"`
	Name      string `json:"name,omitempty"`
	Input     any    `json:"input,omitempty"`
	ToolUseID string `json:"tool_use_id,omitempty"`
	Content   string `json:"content,omitempty"`
}

// toolParam ツール定義
type toolParam struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	InputSchema any    `json:"input_schema"`
}

// streamEvent SSE イベント
type streamEvent struct {
	Type         string        `json:"type"`
	Index        int           `json:"index,omitempty"`
	ContentBlock *contentBlock `json:"content_block,omitempty"`
	Delta        *deltaBlock   `json:"delta,omitempty"`
	Message      *messageResp  `json:"message,omitempty"`
	Usage        *usageInfo    `json:"usage,omitempty"`
}

// usageInfo Token 使用量
type usageInfo struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// contentBlock コンテンツブロック
type contentBlock struct {
	Type  string `json:"type"`
	ID    string `json:"id,omitempty"`
	Name  string `json:"name,omitempty"`
	Text  string `json:"text,omitempty"`
	Input any    `json:"input,omitempty"`
}

// deltaBlock デルタブロック
type deltaBlock struct {
	Type        string `json:"type"`
	Text        string `json:"text,omitempty"`
	PartialJSON string `json:"partial_json,omitempty"`
}

// messageResp メッセージレスポンス
type messageResp struct {
	ID           string `json:"id"`
	Type         string `json:"type"`
	Role         string `json:"role"`
	StopReason   string `json:"stop_reason,omitempty"`
	StopSequence string `json:"stop_sequence,omitempty"`
}

// ==================== LLMClient Implementation ====================

// ChatStream 发送流式对话请求
func (c *Client) ChatStream(ctx context.Context, req *llm.Request) (<-chan *llm.Chunk, error) {
	// リクエスト構築
	apiReq := messagesRequest{
		Model:     c.model,
		MaxTokens: maxTokens,
		Stream:    true,
	}

	// システムプロンプト
	if req.SystemPrompt != "" {
		apiReq.System = req.SystemPrompt
	}

	// メッセージ変換
	apiReq.Messages = convertMessages(req.Messages)

	// ツール変換
	if len(req.Tools) > 0 {
		apiReq.Tools = convertTools(req.Tools)
	}

	// JSON エンコード
	body, err := json.Marshal(apiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// HTTP リクエスト作成
	httpReq, err := http.NewRequestWithContext(ctx, "POST", apiEndpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("anthropic-version", apiVersion)

	// デバッグ: リクエスト内容をログ
	log.Printf("[Anthropic] Request: messages=%d, tools=%d", len(apiReq.Messages), len(apiReq.Tools))
	// メッセージの role 順序をログ
	var roles []string
	for _, m := range apiReq.Messages {
		roles = append(roles, m.Role)
	}
	log.Printf("[Anthropic] Message roles: %v", roles)

	// リクエスト送信
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		log.Printf("[Anthropic] API Error: %d - %s", resp.StatusCode, string(body))
		return nil, fmt.Errorf("anthropic API error: %d - %s", resp.StatusCode, string(body))
	}

	log.Printf("[Anthropic] Response status: %d", resp.StatusCode)

	// ストリーム読み取り開始
	ch := make(chan *llm.Chunk, 32)
	go func() {
		defer close(ch)
		defer resp.Body.Close()
		c.readStream(ctx, resp.Body, ch)
	}()

	return ch, nil
}

// readStream SSE ストリームを読み取り
func (c *Client) readStream(ctx context.Context, body io.Reader, ch chan<- *llm.Chunk) {
	scanner := bufio.NewScanner(body)

	// 現在処理中のツール呼び出し
	var currentToolCall *llm.ToolCall
	var toolInputBuffer strings.Builder
	var lastUsage *llm.Usage

	for scanner.Scan() {
		line := scanner.Text()

		// SSE データ行のみ処理
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			ch <- &llm.Chunk{Type: llm.ChunkDone, Usage: lastUsage}
			return
		}

		var event streamEvent
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			log.Printf("[Anthropic] JSON parse error: %v, data: %s", err, data)
			continue
		}

		// 记录 usage（message_start 和 message_delta 都可能包含 usage）
		if event.Usage != nil {
			lastUsage = &llm.Usage{
				InputTokens:  event.Usage.InputTokens,
				OutputTokens: event.Usage.OutputTokens,
			}
		}

		switch event.Type {
		case "content_block_start":
			if event.ContentBlock != nil && event.ContentBlock.Type == "tool_use" {
				// ツール呼び出し開始
				currentToolCall = &llm.ToolCall{
					ID:   event.ContentBlock.ID,
					Name: event.ContentBlock.Name,
				}
				toolInputBuffer.Reset()
			}

		case "content_block_delta":
			if event.Delta == nil {
				continue
			}

			switch event.Delta.Type {
			case "text_delta":
				// テキストデルタ
				if event.Delta.Text != "" {
					select {
					case ch <- &llm.Chunk{Type: llm.ChunkText, Content: event.Delta.Text}:
					case <-ctx.Done():
						ch <- &llm.Chunk{Type: llm.ChunkError, Error: ctx.Err()}
						return
					}
				}

			case "input_json_delta":
				// ツール入力 JSON デルタ
				if currentToolCall != nil && event.Delta.PartialJSON != "" {
					toolInputBuffer.WriteString(event.Delta.PartialJSON)
				}
			}

		case "content_block_stop":
			// ツール呼び出し完了
			if currentToolCall != nil {
				currentToolCall.Params = toolInputBuffer.String()
				select {
				case ch <- &llm.Chunk{Type: llm.ChunkToolCall, ToolCall: currentToolCall}:
				case <-ctx.Done():
					ch <- &llm.Chunk{Type: llm.ChunkError, Error: ctx.Err()}
					return
				}
				currentToolCall = nil
				toolInputBuffer.Reset()
			}

		case "message_stop":
			ch <- &llm.Chunk{Type: llm.ChunkDone, Usage: lastUsage}
			return

		case "error":
			log.Printf("[Anthropic] Stream error event: %+v", event)
			ch <- &llm.Chunk{Type: llm.ChunkError, Error: fmt.Errorf("stream error")}
			return
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("[Anthropic] Scanner error: %v", err)
		ch <- &llm.Chunk{Type: llm.ChunkError, Error: err}
	}
}

// Close 关闭客户端
func (c *Client) Close() error {
	return nil
}

// ==================== Helpers ====================

// convertMessages llm.Message を Anthropic 形式に変換
func convertMessages(msgs []llm.Message) []messageParam {
	var result []messageParam

	for _, msg := range msgs {
		switch msg.Role {
		case "user":
			result = append(result, messageParam{
				Role: "user",
				Content: []contentPart{
					{Type: "text", Text: msg.Content},
				},
			})

		case "assistant":
			var parts []contentPart
			if msg.Content != "" {
				parts = append(parts, contentPart{Type: "text", Text: msg.Content})
			}
			for _, tc := range msg.ToolCalls {
				var input any
				json.Unmarshal([]byte(tc.Params), &input)
				parts = append(parts, contentPart{
					Type:  "tool_use",
					ID:    tc.ID,
					Name:  tc.Name,
					Input: input,
				})
			}
			if len(parts) > 0 {
				result = append(result, messageParam{
					Role:    "assistant",
					Content: parts,
				})
			}

		case "tool":
			if msg.ToolResult != nil {
				result = append(result, messageParam{
					Role: "user",
					Content: []contentPart{
						{
							Type:      "tool_result",
							ToolUseID: msg.ToolResult.CallID,
							Content:   msg.ToolResult.Content,
						},
					},
				})
			}
		}
	}

	return result
}

// convertTools llm.ToolDefinition を Anthropic 形式に変換
func convertTools(tools []llm.ToolDefinition) []toolParam {
	var result []toolParam
	for _, t := range tools {
		var schema any
		json.Unmarshal(t.Parameters, &schema)
		result = append(result, toolParam{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: schema,
		})
	}
	return result
}

// atlhyper_master_v2/ai/llm/interfaces.go
// LLM 抽象层接口定义
package llm

import (
	"context"
	"encoding/json"
)

// LLMClient LLM 客户端接口
// 所有 LLM 提供商（Gemini, OpenAI 等）实现此接口
type LLMClient interface {
	// ChatStream 发送流式对话请求
	// 返回 Chunk channel，调用方通过 range 读取流式响应
	// channel 关闭表示响应结束
	ChatStream(ctx context.Context, req *Request) (<-chan *Chunk, error)

	// Close 关闭客户端，释放资源
	Close() error
}

// Request 对话请求
type Request struct {
	SystemPrompt string           // 系统提示词
	Messages     []Message        // 历史消息
	Tools        []ToolDefinition // Function Calling 定义
}

// Message 消息
type Message struct {
	Role       string      // user / assistant / tool
	Content    string      // 文本内容
	ToolCalls  []ToolCall  // assistant 发起的 tool calls
	ToolResult *ToolResult // tool 执行结果（role=tool 时使用）
}

// ToolCall LLM 发起的工具调用
type ToolCall struct {
	ID     string // tool call ID
	Name   string // 函数名
	Params string // JSON 参数
}

// ToolResult 工具执行结果
type ToolResult struct {
	CallID  string // 对应的 tool call ID
	Name    string // 函数名
	Content string // 执行结果
}

// ToolDefinition 工具定义（Function Calling Schema）
type ToolDefinition struct {
	Name        string          // 函数名
	Description string          // 描述
	Parameters  json.RawMessage // JSON Schema
}

// Chunk 流式响应块
type Chunk struct {
	Type     ChunkType // 类型
	Content  string    // 文本片段（Type=ChunkText 时使用）
	ToolCall *ToolCall // Tool Call 信息（Type=ChunkToolCall 时使用）
	Error    error     // 错误信息（Type=ChunkError 时使用）
	Usage    *Usage    // Token 使用量（Type=ChunkDone 时可能返回）
}

// Usage Token 使用量
type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// ChunkType 响应块类型
type ChunkType string

const (
	ChunkText     ChunkType = "text"      // 文本片段
	ChunkToolCall ChunkType = "tool_call" // 工具调用
	ChunkDone     ChunkType = "done"      // 响应结束
	ChunkError    ChunkType = "error"     // 错误
)

// Config LLM 配置
type Config struct {
	Provider string // gemini / openai
	APIKey   string // API Key
	Model    string // 模型名称 (e.g. gemini-2.0-flash)
}

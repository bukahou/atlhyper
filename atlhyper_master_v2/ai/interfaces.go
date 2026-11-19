// atlhyper_master_v2/ai/interfaces.go
// AIService 对外接口 + 类型定义
package ai

import (
	"context"
	"time"

	"AtlHyper/atlhyper_master_v2/ai/llm"
	"AtlHyper/atlhyper_master_v2/ai/toolkit"
)

// MaxToolCallsPerRound 对话与深度分析每轮最多执行的工具调用数（提示词从这里渲染）
const MaxToolCallsPerRound = 5

// PermissionCheck 判断会看到工具结果的人是否拥有某项权限（对话里是调用者本人，深度分析是能看报告的人）
type PermissionCheck = toolkit.PermissionCheck

// AIService AI 服务接口
type AIService interface {
	// CreateConversation 创建对话
	CreateConversation(ctx context.Context, userID int64, clusterID, title string) (*Conversation, error)

	// Chat 发送消息并获取流式响应
	// 返回 ChatChunk channel，通过 SSE 推送给前端
	Chat(ctx context.Context, req *ChatRequest) (<-chan *ChatChunk, error)

	// GetConversations 获取用户对话列表
	GetConversations(ctx context.Context, userID int64, limit, offset int) ([]*Conversation, error)

	// GetMessages 获取对话历史消息（只限会话创建者）
	GetMessages(ctx context.Context, userID, conversationID int64) ([]*Message, error)

	// DeleteConversation 删除对话及其所有消息
	DeleteConversation(ctx context.Context, userID, conversationID int64) error

	// Tools 工具注册表：启动时由装配方注册（ai/tools），运行中只读
	Tools() *toolkit.Registry

	// Analyze 非交互式多轮 Tool Calling 分析（后台执行，无 SSE）
	Analyze(ctx context.Context, req *AnalyzeRequest) (*AnalyzeResult, error)

	// Complete 单轮 LLM 调用（无 Tool，用于 background 摘要等）
	Complete(ctx context.Context, req *CompleteRequest) (*CompleteResult, error)
}

// ChatRequest 发送消息请求
type ChatRequest struct {
	ConversationID int64  // 对话 ID
	ClusterID      string // 目标集群
	UserID         int64  // 用户 ID（只能续聊自己的会话）
	Message        string // 用户消息

	// Allow 调用者拥有的权限：工具清单与工具执行都按它过滤；nil = 一律拒绝
	Allow PermissionCheck
}

// ChatChunk SSE 流式响应块
type ChatChunk struct {
	Type string `json:"type"` // text / tool_call / tool_result / notice / done / error
	// Round 第几次调模型（从 1 开始）；text / tool_call / tool_result / notice 都带，前端按它分轮，⛔ 不按文字顺序猜
	Round   int    `json:"round,omitempty"`
	Content string `json:"content,omitempty"` // 文本内容 / 工具结果 / 错误说明
	// CallID 模型给的调用 ID：tool_call 与 tool_result 按它配对（同一轮可能并行调用同名工具）
	CallID  string     `json:"callId,omitempty"`
	Tool    string     `json:"tool,omitempty"`    // tool 名称
	Params  string     `json:"params,omitempty"`  // tool 参数 JSON
	IsError bool       `json:"isError,omitempty"` // tool_result：执行失败
	Notice  string     `json:"notice,omitempty"`  // notice：代码（NoticeHistoryTrimmed / NoticeToolCallsCapped），文案前端出
	Stats   *ChatStats `json:"stats,omitempty"`   // 统计信息（done 时返回）
}

// 流里的提示代码（文案由前端按语言出，⛔ 不以正文文字混进回答）
const (
	NoticeHistoryTrimmed  = "historyTrimmed"  // 历史消息已按轮裁剪以适应模型上下文
	NoticeToolCallsCapped = "toolCallsCapped" // 本轮工具调用超过上限，只执行了前面几个
)

// ChatStats 对话统计信息
type ChatStats struct {
	Rounds         int `json:"rounds"`         // 思考轮次（AI 调用次数）
	TotalToolCalls int `json:"totalToolCalls"` // 总指令数（所有轮次的 Tool 调用总数）
	InputTokens    int `json:"inputTokens"`    // 输入 Token 数
	OutputTokens   int `json:"outputTokens"`   // 输出 Token 数
}

// Conversation 对话
type Conversation struct {
	ID           int64  `json:"id"`
	UserID       int64  `json:"userId"`
	ClusterID    string `json:"clusterId"`
	Title        string `json:"title"`
	MessageCount int    `json:"messageCount"`
	// 累计统计
	TotalInputTokens  int64     `json:"totalInputTokens"`  // 累计输入 Token
	TotalOutputTokens int64     `json:"totalOutputTokens"` // 累计输出 Token
	TotalToolCalls    int       `json:"totalToolCalls"`    // 累计指令数
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

// Message 消息（工具调用与结果是结构化字段，⛔ 不以 JSON 字符串下发让前端再解析）
type Message struct {
	ID             int64  `json:"id"`
	ConversationID int64  `json:"conversationId"`
	Role           string `json:"role"` // user / assistant / tool
	// Content user / assistant 的正文；tool 消息为空（结果在 ToolResult）
	Content    string          `json:"content"`
	ToolCalls  []llm.ToolCall  `json:"toolCalls,omitempty"`  // assistant 发起的工具调用（一条带调用的 assistant 消息 = 一轮）
	ToolResult *llm.ToolResult `json:"toolResult,omitempty"` // tool 消息：按 callId 对应到调用
	CreatedAt  time.Time       `json:"createdAt"`
}

// ==================== Analyze（多轮 Tool Calling）====================

// AnalyzeRequest 非交互式分析请求
type AnalyzeRequest struct {
	ClusterID    string        // 目标集群
	Role         string        // 角色名（用于预算扣减，如 "analysis"）
	SystemPrompt string        // 调用方提供的系统提示词
	UserPrompt   string        // 调用方构建的用户提示词
	MaxRounds    int           // 最大 Tool 调用轮次（默认 8）
	Timeout      time.Duration // 全局超时（默认 5min）

	// Allow 分析报告读者拥有的权限：工具清单与工具执行都按它过滤；nil = 一律拒绝
	Allow PermissionCheck
}

// AnalyzeResult 分析结果
type AnalyzeResult struct {
	Response     string        // LLM 最终文本输出
	ToolCalls    int           // 总 Tool 调用次数
	InputTokens  int           // 总输入 Token
	OutputTokens int           // 总输出 Token
	ProviderName string        // Provider 名称
	Model        string        // 模型名称
	Steps        []AnalyzeStep // 调查步骤记录
	Duration     time.Duration // 分析总耗时（含各轮 LLM 调用与工具执行）
}

// AnalyzeStep 单轮调查步骤
type AnalyzeStep struct {
	Round     int              `json:"round"`
	Thinking  string           `json:"thinking"`
	ToolCalls []ToolCallRecord `json:"toolCalls"`
}

// ToolCallRecord Tool 调用记录
type ToolCallRecord struct {
	Tool          string `json:"tool"`
	Params        string `json:"params"`
	ResultSummary string `json:"resultSummary"`
}

// ==================== Complete（单轮 LLM 调用）====================

// CompleteRequest 单轮 LLM 调用请求（无 Tool）
type CompleteRequest struct {
	Role         string // 角色名（用于角色路由、预算扣减，如 "background"）
	SystemPrompt string
	UserPrompt   string
}

// CompleteResult 单轮 LLM 调用结果
type CompleteResult struct {
	Response     string
	InputTokens  int
	OutputTokens int
	ProviderID   int64         // Provider ID（供调用方记录）
	ProviderName string        // Provider 名称（供调用方日志）
	Model        string        // 模型名称
	Duration     time.Duration // LLM 调用耗时
}

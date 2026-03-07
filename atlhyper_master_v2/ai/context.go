// atlhyper_master_v2/ai/context.go
// 上下文管理器
// 根据 Provider 的 context_window 在发送前裁剪消息，防止静默截断
package ai

import (
	"AtlHyper/atlhyper_master_v2/ai/llm"
)

// ContextManager 上下文管理器
type ContextManager struct {
	contextWindow int // Provider 上下文窗口 (tokens), 0 = 不裁剪
	outputReserve int // 为输出保留的 token 数
}

// NewContextManager 创建上下文管理器
func NewContextManager(contextWindow int) *ContextManager {
	if contextWindow <= 0 {
		return &ContextManager{contextWindow: 0}
	}
	// 为输出保留 ~20% 的上下文
	reserve := contextWindow / 5
	if reserve < 1000 {
		reserve = 1000
	}
	return &ContextManager{
		contextWindow: contextWindow,
		outputReserve: reserve,
	}
}

// FitMessages 裁剪消息列表以适应上下文窗口
// 策略: 从最新消息往回填充，保证最近的对话完整
// 返回: 裁剪后的消息列表 + 是否发生了裁剪
func (cm *ContextManager) FitMessages(systemPrompt string, messages []llm.Message) ([]llm.Message, bool) {
	if cm.contextWindow <= 0 {
		return messages, false
	}

	budget := cm.contextWindow - cm.outputReserve - estimateTokens(systemPrompt)
	if budget <= 0 {
		return messages, true
	}

	// 从最新消息往前，贪心填充
	var fitted []llm.Message
	for i := len(messages) - 1; i >= 0; i-- {
		cost := estimateMessageTokens(&messages[i])
		if budget-cost < 0 && len(fitted) > 0 {
			break
		}
		budget -= cost
		fitted = append([]llm.Message{messages[i]}, fitted...)
	}

	truncated := len(fitted) < len(messages)
	return fitted, truncated
}

// estimateTokens 估算文本的 token 数
// 粗估: 英文 ~4 chars/token, 中文 ~1.5 chars/token
// 取保守值: 1 token ≈ 2.5 字符
func estimateTokens(text string) int {
	chars := len([]rune(text))
	return (chars*10 + 24) / 25 // 等价于 chars / 2.5 向上取整
}

// estimateMessageTokens 估算单条消息的 token 数
func estimateMessageTokens(msg *llm.Message) int {
	tokens := estimateTokens(msg.Content)
	for _, tc := range msg.ToolCalls {
		tokens += estimateTokens(tc.Params) + 20
	}
	if msg.ToolResult != nil {
		tokens += estimateTokens(msg.ToolResult.Content) + 20
	}
	return tokens + 5 // 消息 overhead
}

// toolResultMaxLen 根据 context_window 决定 Tool 结果最大字符数
func toolResultMaxLen(contextWindow int) int {
	if contextWindow <= 0 {
		return 32000
	}
	if contextWindow <= 8192 {
		return 2000
	}
	if contextWindow <= 32768 {
		return 8000
	}
	return 32000
}

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

// FitMessages 裁剪消息列表以适应上下文窗口。
// 以「一问一答」为单位（一条用户消息 + 其后的助手 / 工具消息）从最旧的整轮丢起：
// ⛔ 不把工具调用与它的结果拆开（拆开后 Claude 等提供商拒收孤立的工具结果）。
// 最新的一轮总是保留（哪怕单独就超了，交给提供商报错，也不拆）。
// 返回: 裁剪后的消息列表 + 是否发生了裁剪
func (cm *ContextManager) FitMessages(systemPrompt string, messages []llm.Message) ([]llm.Message, bool) {
	if cm.contextWindow <= 0 {
		return messages, false
	}
	budget := cm.contextWindow - cm.outputReserve - estimateTokens(systemPrompt)
	starts := turnStarts(len(messages), func(i int) string { return messages[i].Role })
	keepFrom := starts[len(starts)-1]
	for i := len(starts) - 1; i >= 0; i-- {
		end := len(messages)
		if i+1 < len(starts) {
			end = starts[i+1]
		}
		cost := 0
		for j := starts[i]; j < end; j++ {
			cost += estimateMessageTokens(&messages[j])
		}
		if budget-cost < 0 && i < len(starts)-1 {
			break
		}
		budget -= cost
		keepFrom = starts[i]
	}
	return messages[keepFrom:], keepFrom > 0
}

// turnStarts 每一轮的起点下标（一轮从一条用户消息开始；开头若不是用户消息，自成一轮）。n 为 0 时返回 [0]
func turnStarts(n int, role func(int) string) []int {
	starts := []int{0}
	for i := 1; i < n; i++ {
		if role(i) == "user" {
			starts = append(starts, i)
		}
	}
	return starts
}

// recentTurnsStart 最多保留 maxMessages 条时的起点：只在一轮的开头切，⛔ 不切进一轮中间；
// 最新一轮比上限还长时整轮保留
func recentTurnsStart(n, maxMessages int, role func(int) string) int {
	starts := turnStarts(n, role)
	for _, st := range starts {
		if n-st <= maxMessages {
			return st
		}
	}
	return starts[len(starts)-1]
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

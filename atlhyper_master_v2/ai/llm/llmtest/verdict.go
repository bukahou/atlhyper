package llmtest

import (
	"encoding/json"
	"fmt"
	"reflect"
	"slices"

	"AtlHyper/atlhyper_master_v2/ai/llm"
)

// 判定只看分段序列、不碰网络：返回违反契约的条目，空 = 符合。

// completedProblems 正常回答：以完成收尾，文字分段按序到达，工具调用拼成完整的一个，完成分段带用量
func completedProblems(chunks []*llm.Chunk, want Reply, toolCallIDOnWire bool) []string {
	end, problems := terminalOf(chunks)
	if end != nil && end.Type != llm.ChunkDone {
		problems = append(problems, fmt.Sprintf("应以完成收尾，实际以错误收尾（分类 %q）：%v", llm.Classify(end.Error), end.Error))
	}
	if texts := textsOf(chunks); !slices.Equal(texts, want.Texts) {
		problems = append(problems, fmt.Sprintf("文字分段为 %q，应按序为 %q", texts, want.Texts))
	}
	problems = append(problems, toolCallProblems(toolCallsOf(chunks), want.ToolCall, toolCallIDOnWire)...)
	if end != nil && end.Type == llm.ChunkDone {
		switch {
		case end.Usage == nil:
			problems = append(problems, "完成分段没带用量")
		case *end.Usage != want.Usage:
			problems = append(problems, fmt.Sprintf("用量为 %+v，应为 %+v", *end.Usage, want.Usage))
		}
	}
	return problems
}

// failedProblems 上游失败：ChatStream 直接报错，或流以错误收尾（⛔ 不能以完成收尾），错误分类在 kinds 之内
func failedProblems(chatErr error, chunks []*llm.Chunk, kinds ...llm.ErrorKind) []string {
	if chatErr != nil {
		if kind := llm.Classify(chatErr); !slices.Contains(kinds, kind) {
			return []string{fmt.Sprintf("ChatStream 的错误分类为 %q，应为 %q：%v", kind, kinds, chatErr)}
		}
		return nil
	}
	end, problems := terminalOf(chunks)
	if end == nil {
		return problems
	}
	if end.Type == llm.ChunkDone {
		return append(problems, "以完成收尾：上游出错或没按约定收尾，回答却会被当成成功")
	}
	if kind := llm.Classify(end.Error); !slices.Contains(kinds, kind) {
		problems = append(problems, fmt.Sprintf("错误分类为 %q，应为 %q：%v", kind, kinds, end.Error))
	}
	return problems
}

// canceledProblems 调用方取消：⛔ 不能以完成收尾；报出的错误必须是取消类。
// 可以不报（调用方已不再读，缓冲区满时适配层只能放弃），但报了就得在最后。
func canceledProblems(chatErr error, chunks []*llm.Chunk) []string {
	if chatErr != nil {
		if kind := llm.Classify(chatErr); kind != llm.ErrKindCanceled {
			return []string{fmt.Sprintf("ChatStream 的错误分类为 %q，应为 canceled：%v", kind, chatErr)}
		}
		return nil
	}
	var problems []string
	for i, c := range chunks {
		switch {
		case c == nil:
			problems = append(problems, fmt.Sprintf("第 %d 段是 nil", i))
		case c.Type == llm.ChunkDone:
			problems = append(problems, "取消后以完成收尾：没说完的回答会被当成完整的")
		case c.Type == llm.ChunkError:
			if kind := llm.Classify(c.Error); kind != llm.ErrKindCanceled {
				problems = append(problems, fmt.Sprintf("取消后报出的错误分类为 %q，应为 canceled：%v", kind, c.Error))
			}
			if i != len(chunks)-1 {
				problems = append(problems, fmt.Sprintf("第 %d 段（错误）之后还有分段：收尾分段必须在最后", i))
			}
		}
	}
	return problems
}

// terminalOf 流的收尾分段：必须恰好一个（完成或错误），而且在最后。
// 通道没收尾就关闭时 ReadStream 会兜底为未完成，但直接读通道的调用方会当成正常结束，所以契约要求适配层自己收尾。
func terminalOf(chunks []*llm.Chunk) (*llm.Chunk, []string) {
	var problems []string
	for i, c := range chunks {
		if c == nil {
			problems = append(problems, fmt.Sprintf("第 %d 段是 nil", i))
			continue
		}
		if isTerminal(c) && i != len(chunks)-1 {
			problems = append(problems, fmt.Sprintf("第 %d 段（%s）之后还有分段：收尾分段必须在最后", i, c.Type))
		}
	}
	if len(chunks) == 0 || chunks[len(chunks)-1] == nil || !isTerminal(chunks[len(chunks)-1]) {
		return nil, append(problems, "通道关闭前没有完成或错误分段：适配层必须自己收尾（ReadStream 会兜底为未完成，直接读通道的调用方会当成正常结束）")
	}
	return chunks[len(chunks)-1], problems
}

func isTerminal(c *llm.Chunk) bool { return c.Type == llm.ChunkDone || c.Type == llm.ChunkError }

func toolCallProblems(got []llm.ToolCall, want *ToolCallReply, toolCallIDOnWire bool) []string {
	if want == nil {
		if len(got) > 0 {
			return []string{fmt.Sprintf("不该有工具调用，实际 %+v", got)}
		}
		return nil
	}
	if len(got) != 1 {
		return []string{fmt.Sprintf("应拼成恰好一个工具调用，实际 %d 个：%+v", len(got), got)}
	}
	call := got[0]
	var problems []string
	if call.Name != want.Name {
		problems = append(problems, fmt.Sprintf("工具名为 %q，应为 %q", call.Name, want.Name))
	}
	switch {
	case toolCallIDOnWire && call.ID != want.ID:
		problems = append(problems, fmt.Sprintf("工具调用 ID 为 %q，应原样透传上游给的 %q", call.ID, want.ID))
	case !toolCallIDOnWire && call.ID == "":
		problems = append(problems, "上游不给工具调用 ID 时，适配层应生成一个非空的 ID")
	}
	if !sameJSON(call.Params, want.Args()) {
		problems = append(problems, fmt.Sprintf("工具参数为 %s，应为 %s", call.Params, want.Args()))
	}
	return problems
}

// sameJSON 两段 JSON 语义相同（键序、空白不计）
func sameJSON(a, b string) bool {
	var left, right any
	if json.Unmarshal([]byte(a), &left) != nil || json.Unmarshal([]byte(b), &right) != nil {
		return false
	}
	return reflect.DeepEqual(left, right)
}

func textsOf(chunks []*llm.Chunk) []string {
	var texts []string
	for _, c := range chunks {
		if c != nil && c.Type == llm.ChunkText {
			texts = append(texts, c.Content)
		}
	}
	return texts
}

func toolCallsOf(chunks []*llm.Chunk) []llm.ToolCall {
	var calls []llm.ToolCall
	for _, c := range chunks {
		if c != nil && c.Type == llm.ChunkToolCall && c.ToolCall != nil {
			calls = append(calls, *c.ToolCall)
		}
	}
	return calls
}

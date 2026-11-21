// atlhyper_master_v2/ai/llm/anthropic/convert.go
// llm 抽象层 ↔ Claude Messages 接口的请求转换
package anthropic

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"

	"AtlHyper/atlhyper_master_v2/ai/llm"

	anthropicsdk "github.com/anthropics/anthropic-sdk-go/v2"
)

// skippedToolResult 原样回放的那一轮里被工具循环截掉的调用：补这条结果（Claude 要求每个 tool_use 都有结果）
const skippedToolResult = "未执行：本轮工具调用过多，已截断"

// buildParams 组装一次请求。思考按模型默认（当前型号为自适应）；不传采样参数（新型号拒收）。
func buildParams(model string, req *llm.Request) (anthropicsdk.MessageNewParams, error) {
	params := anthropicsdk.MessageNewParams{
		Model:     anthropicsdk.Model(model),
		MaxTokens: maxOutputTokens,
		Messages:  convertMessages(req.Messages),
		// 请求级自动缓存：断点落在最后一个可缓存块上，工具循环的下一轮从缓存读之前的全部内容
		CacheControl: anthropicsdk.NewCacheControlEphemeralParam(),
	}
	if req.SystemPrompt != "" {
		// 断点：工具定义 + 系统提示词（同一角色的请求之间不变）
		params.System = []anthropicsdk.TextBlockParam{{
			Text:         req.SystemPrompt,
			CacheControl: anthropicsdk.NewCacheControlEphemeralParam(),
		}}
	}

	tools := req.Tools
	if len(tools) == 0 {
		// 这一轮不给工具（例如最后一轮要求直接作答），但历史里有工具往来：
		// Claude 要求含 tool_use / tool_result 的请求带上工具定义 ⇒ 按名字补上，并声明这一轮不许调用
		if seen := toolsSeenIn(req.Messages); len(seen) > 0 {
			tools = seen
			params.ToolChoice = anthropicsdk.ToolChoiceUnionParam{OfNone: &anthropicsdk.ToolChoiceNoneParam{}}
		}
	}
	if len(tools) > 0 {
		converted, err := convertTools(tools)
		if err != nil {
			return params, err
		}
		params.Tools = converted
	}
	return params, nil
}

// ==================== 消息 ====================

// validToolUseID Claude 只认这种工具调用 ID
var validToolUseID = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// convertMessages 把 llm 消息转成 Claude 的消息：
//   - 同一轮的工具结果并进一条用户消息；连续同角色的消息合并成一条；开头的 assistant 消息去掉（须从用户消息开始）
//   - 工具调用与结果配对：没有结果的调用去掉，找不到调用的结果改成文字（历史按条数截断后会出现，Claude 直接 400）
//   - 不合法的工具调用 ID 重写，调用与结果同步改
//   - 带着本 provider ProviderTurn 的 assistant 消息原样回放（思考块带签名，不能改）；
//     其中没有结果的工具调用（被工具循环截掉的）补一条「未执行」的结果
func convertMessages(msgs []llm.Message) []anthropicsdk.MessageParam {
	b := &messageBuilder{ids: map[string]string{}, used: map[string]bool{}}
	answered := answeredToolCalls(msgs)

	var pending []string // 原样回放的那一轮里还没等到结果的调用
	flushPending := func() {
		for _, id := range pending {
			b.addUser(anthropicsdk.NewToolResultBlock(id, skippedToolResult, true))
		}
		pending = nil
	}

	for index, msg := range msgs {
		if msg.Role != "tool" {
			flushPending()
		}
		switch msg.Role {
		case "assistant":
			if turn, ok := replayableTurn(msg.ProviderTurn); ok {
				b.addAssistant(turn.Content...)
				for _, id := range toolUseIDsOf(turn.Content) {
					b.keepID(id)
					if !answered[index][id] {
						pending = append(pending, id)
					}
				}
				continue
			}
			var blocks []anthropicsdk.ContentBlockParamUnion
			if msg.Content != "" {
				blocks = append(blocks, anthropicsdk.NewTextBlock(msg.Content))
			}
			for _, call := range msg.ToolCalls {
				if !answered[index][call.ID] {
					continue
				}
				blocks = append(blocks, anthropicsdk.NewToolUseBlock(b.idFor(call.ID), toolInput(call), call.Name))
			}
			b.addAssistant(blocks...)

		case "tool":
			if msg.ToolResult == nil {
				continue
			}
			result := msg.ToolResult
			content := result.Content
			if content == "" {
				content = "（无输出）"
			}
			if pairedWithCall(msgs, index) {
				b.addUser(anthropicsdk.NewToolResultBlock(b.idFor(result.CallID), content, result.IsError))
				pending = removeID(pending, result.CallID)
			} else {
				b.addUser(anthropicsdk.NewTextBlock(fmt.Sprintf("（工具 %s 的结果）%s", result.Name, content)))
			}

		case "user":
			if msg.Content != "" {
				b.addUser(anthropicsdk.NewTextBlock(msg.Content))
			}
		}
	}
	flushPending()
	return b.dropLeadingAssistant()
}

// messageBuilder 逐条追加消息，连续同角色合并；维护工具调用 ID 的重写表
type messageBuilder struct {
	messages []anthropicsdk.MessageParam
	ids      map[string]string // 原 ID → 发给 Claude 的 ID
	used     map[string]bool   // 已占用的 ID（防止重写出的 ID 撞上原有的）
}

func (b *messageBuilder) addUser(blocks ...anthropicsdk.ContentBlockParamUnion) {
	if len(blocks) == 0 {
		return
	}
	if last := len(b.messages) - 1; last >= 0 && b.messages[last].OfUser != nil {
		b.messages[last].OfUser.Content = append(b.messages[last].OfUser.Content, blocks...)
		return
	}
	b.messages = append(b.messages, anthropicsdk.NewUserMessage(blocks...))
}

func (b *messageBuilder) addAssistant(blocks ...anthropicsdk.ContentBlockParamUnion) {
	if len(blocks) == 0 {
		return
	}
	if last := len(b.messages) - 1; last >= 0 && b.messages[last].OfAssistant != nil {
		b.messages[last].OfAssistant.Content = append(b.messages[last].OfAssistant.Content, blocks...)
		return
	}
	b.messages = append(b.messages, anthropicsdk.NewAssistantMessage(blocks...))
}

// keepID 登记一个原样使用的 ID（回放的那一轮由 Claude 生成，必然合法）
func (b *messageBuilder) keepID(id string) {
	b.ids[id] = id
	b.used[id] = true
}

// idFor 返回发给 Claude 的 ID：合法的原样用，不合法的重写成 call_N（同一原 ID 始终映射到同一个）
func (b *messageBuilder) idFor(original string) string {
	if id, ok := b.ids[original]; ok {
		return id
	}
	id := original
	if !validToolUseID.MatchString(id) || b.used[id] {
		for n := len(b.ids); ; n++ {
			id = "call_" + strconv.Itoa(n)
			if !b.used[id] {
				break
			}
		}
	}
	b.ids[original] = id
	b.used[id] = true
	return id
}

// dropLeadingAssistant 去掉开头的 assistant 消息（历史截断后可能以它开头）
func (b *messageBuilder) dropLeadingAssistant() []anthropicsdk.MessageParam {
	out := b.messages
	for len(out) > 0 && out[0].OfAssistant != nil {
		out = out[1:]
	}
	return out
}

// replayableTurn 取出本 provider 产生的整轮回答（别的 provider 留下的类型对不上，按普通消息处理）
func replayableTurn(turn any) (*anthropicsdk.AssistantMessageParam, bool) {
	param, ok := turn.(anthropicsdk.MessageParam)
	if !ok || param.OfAssistant == nil || len(param.OfAssistant.Content) == 0 {
		return nil, false
	}
	return param.OfAssistant, true
}

func toolUseIDsOf(blocks []anthropicsdk.ContentBlockParamUnion) []string {
	var ids []string
	for _, block := range blocks {
		if block.OfToolUse != nil {
			ids = append(ids, block.OfToolUse.ID)
		}
	}
	return ids
}

// answeredToolCalls 每条 assistant 消息里「后面紧跟着结果」的调用：answered[下标][调用 ID]。
// 结果必须紧跟在调用之后（中间只能是别的工具结果）。
func answeredToolCalls(msgs []llm.Message) map[int]map[string]bool {
	answered := map[int]map[string]bool{}
	for index, msg := range msgs {
		if msg.Role != "assistant" {
			continue
		}
		set := map[string]bool{}
		for next := index + 1; next < len(msgs) && msgs[next].Role == "tool"; next++ {
			if r := msgs[next].ToolResult; r != nil {
				set[r.CallID] = true
			}
		}
		answered[index] = set
	}
	return answered
}

// pairedWithCall 判断 msgs[index]（工具结果）是否紧跟在发起它的那条 assistant 消息之后
func pairedWithCall(msgs []llm.Message, index int) bool {
	id := msgs[index].ToolResult.CallID
	for previous := index - 1; previous >= 0; previous-- {
		switch msgs[previous].Role {
		case "tool":
			continue
		case "assistant":
			if turn, ok := replayableTurn(msgs[previous].ProviderTurn); ok {
				for _, use := range toolUseIDsOf(turn.Content) {
					if use == id {
						return true
					}
				}
				return false
			}
			for _, call := range msgs[previous].ToolCalls {
				if call.ID == id {
					return true
				}
			}
			return false
		default:
			return false
		}
	}
	return false
}

func removeID(ids []string, id string) []string {
	out := ids[:0]
	for _, v := range ids {
		if v != id {
			out = append(out, v)
		}
	}
	return out
}

// toolInput 工具调用参数（JSON 对象）；解析不了时用空对象（Claude 要求 input 是对象）
func toolInput(call llm.ToolCall) any {
	var input map[string]any
	if err := json.Unmarshal([]byte(call.Params), &input); err != nil || input == nil {
		if call.Params != "" && call.Params != "{}" {
			log.Warn("工具调用参数解析失败，按空参数发送", "tool", call.Name, "err", err)
		}
		return map[string]any{}
	}
	return input
}

// ==================== 工具 ====================

// toolsSeenIn 历史里出现过的工具（只有名字）：这一轮不给工具、但历史里有工具往来时补上定义用
func toolsSeenIn(msgs []llm.Message) []llm.ToolDefinition {
	seen := map[string]bool{}
	var defs []llm.ToolDefinition
	add := func(name string) {
		if name != "" && !seen[name] {
			seen[name] = true
			defs = append(defs, llm.ToolDefinition{Name: name})
		}
	}
	for _, msg := range msgs {
		for _, call := range msg.ToolCalls {
			add(call.Name)
		}
		if turn, ok := replayableTurn(msg.ProviderTurn); ok {
			for _, block := range turn.Content {
				if block.OfToolUse != nil {
					add(block.OfToolUse.Name)
				}
			}
		}
	}
	return defs
}

// convertTools 转换工具定义（参数是 JSON Schema：properties / required 放进对应字段，其余原样带上）
func convertTools(tools []llm.ToolDefinition) ([]anthropicsdk.ToolUnionParam, error) {
	converted := make([]anthropicsdk.ToolUnionParam, 0, len(tools))
	for _, tool := range tools {
		schema := anthropicsdk.ToolInputSchemaParam{}
		if len(tool.Parameters) > 0 {
			var fields map[string]any
			if err := json.Unmarshal(tool.Parameters, &fields); err != nil {
				return nil, fmt.Errorf("工具 %s 的参数定义不是合法 JSON: %w", tool.Name, err)
			}
			for key, value := range fields {
				switch key {
				case "type": // 固定是 object
				case "properties":
					schema.Properties = value
				case "required":
					list, _ := value.([]any)
					for _, name := range list {
						if text, ok := name.(string); ok {
							schema.Required = append(schema.Required, text)
						}
					}
				default:
					if schema.ExtraFields == nil {
						schema.ExtraFields = map[string]any{}
					}
					schema.ExtraFields[key] = value
				}
			}
		}
		if schema.Properties == nil {
			// 没有参数的工具（以及按历史补上的工具）也要有 input_schema：全空时 SDK 会省掉这个字段，Claude 回 400
			schema.Properties = map[string]any{}
		}
		wire := anthropicsdk.ToolParam{Name: tool.Name, InputSchema: schema}
		if tool.Description != "" {
			wire.Description = anthropicsdk.String(tool.Description)
		}
		converted = append(converted, anthropicsdk.ToolUnionParam{OfTool: &wire})
	}
	return converted, nil
}

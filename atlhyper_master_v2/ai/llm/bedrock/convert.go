// atlhyper_master_v2/ai/llm/bedrock/convert.go
// llm.Request → Bedrock ConverseStream 输入
package bedrock

import (
	"encoding/json"
	"fmt"

	"AtlHyper/atlhyper_master_v2/ai/llm"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/document"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
)

// emptyObjectSchema 无参数工具的 schema（Bedrock 要求 inputSchema 必填）
var emptyObjectSchema = map[string]any{"type": "object", "properties": map[string]any{}}

// buildInput 组装 ConverseStream 请求
//
// Bedrock 的约束（违反即 ValidationException）：
//   - 首条消息必须是 user；user / assistant 必须交替
//   - 一轮里 assistant 的全部 toolUse 的结果，要放进紧随其后的同一条 user 消息
//   - 内容块不得为空
func buildInput(model string, req *llm.Request) (*bedrockruntime.ConverseStreamInput, error) {
	in := &bedrockruntime.ConverseStreamInput{ModelId: aws.String(model)}

	if req.SystemPrompt != "" {
		in.System = []types.SystemContentBlock{&types.SystemContentBlockMemberText{Value: req.SystemPrompt}}
	}

	msgs, err := convertMessages(req.Messages)
	if err != nil {
		return nil, err
	}
	in.Messages = msgs

	if len(req.Tools) > 0 {
		tools := make([]types.Tool, 0, len(req.Tools))
		for _, td := range req.Tools {
			schema, err := parseSchema(td.Parameters)
			if err != nil {
				return nil, fmt.Errorf("工具 %s 的参数 schema 非法: %w", td.Name, err)
			}
			tools = append(tools, &types.ToolMemberToolSpec{Value: types.ToolSpecification{
				Name:        aws.String(td.Name),
				Description: aws.String(td.Description),
				InputSchema: &types.ToolInputSchemaMemberJson{Value: document.NewLazyDocument(schema)},
			}})
		}
		in.ToolConfig = &types.ToolConfiguration{Tools: tools}
	}
	return in, nil
}

func parseSchema(raw json.RawMessage) (any, error) {
	if len(raw) == 0 {
		return emptyObjectSchema, nil
	}
	var schema any
	if err := json.Unmarshal(raw, &schema); err != nil {
		return nil, err
	}
	return schema, nil
}

// convertMessages 转换消息：相邻同角色合并（tool 结果按 user 计）
// 第一条用户提问之前的一切（裁剪留下的 assistant / tool 结果）都丢弃：
// 首条必须是 user，且孤立的 toolResult（找不到对应 toolUse）会被拒绝
func convertMessages(src []llm.Message) ([]types.Message, error) {
	var out []types.Message
	appendBlocks := func(role types.ConversationRole, blocks []types.ContentBlock) {
		if len(blocks) == 0 || len(out) == 0 {
			return
		}
		if n := len(out); n > 0 && out[n-1].Role == role {
			out[n-1].Content = append(out[n-1].Content, blocks...)
			return
		}
		out = append(out, types.Message{Role: role, Content: blocks})
	}

	for _, m := range src {
		switch m.Role {
		case "user":
			if m.Content == "" {
				continue
			}
			block := []types.ContentBlock{&types.ContentBlockMemberText{Value: m.Content}}
			if len(out) == 0 {
				out = append(out, types.Message{Role: types.ConversationRoleUser, Content: block})
				continue
			}
			appendBlocks(types.ConversationRoleUser, block)
		case "assistant":
			var blocks []types.ContentBlock
			if m.Content != "" {
				blocks = append(blocks, &types.ContentBlockMemberText{Value: m.Content})
			}
			for _, tc := range m.ToolCalls {
				input, err := parseToolInput(tc.Params)
				if err != nil {
					return nil, fmt.Errorf("tool call %s 的参数非法: %w", tc.ID, err)
				}
				blocks = append(blocks, &types.ContentBlockMemberToolUse{Value: types.ToolUseBlock{
					ToolUseId: aws.String(tc.ID),
					Name:      aws.String(tc.Name),
					Input:     document.NewLazyDocument(input),
				}})
			}
			appendBlocks(types.ConversationRoleAssistant, blocks)
		case "tool":
			if m.ToolResult == nil {
				continue
			}
			appendBlocks(types.ConversationRoleUser, []types.ContentBlock{&types.ContentBlockMemberToolResult{Value: types.ToolResultBlock{
				ToolUseId: aws.String(m.ToolResult.CallID),
				Content:   []types.ToolResultContentBlock{&types.ToolResultContentBlockMemberText{Value: m.ToolResult.Content}},
			}}})
		}
	}
	return out, nil
}

func parseToolInput(params string) (any, error) {
	if params == "" {
		return map[string]any{}, nil
	}
	var v any
	if err := json.Unmarshal([]byte(params), &v); err != nil {
		return nil, err
	}
	return v, nil
}

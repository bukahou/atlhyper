// atlhyper_master_v2/ai/llm/bedrock/stream.go
// ConverseStream 事件 → llm.Chunk
package bedrock

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"AtlHyper/atlhyper_master_v2/ai/llm"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
)

// pendingTool 进行中的工具调用：入参以 JSON 片段分多次到达
type pendingTool struct {
	id, name string
	input    strings.Builder
}

// pump 读取事件直到流结束，转换后写入 out（不关闭 out）
//
//   - 文本 delta → ChunkText；推理内容（reasoning）不进正文
//   - 工具入参按 contentBlockIndex 拼接，在 contentBlockStop 时校验 JSON 并发出 ChunkToolCall
//   - 流正常结束 → ChunkDone（带 usage）；出错 → ChunkError 且不发 done
func pump(ctx context.Context, events <-chan types.ConverseStreamOutput, streamErr func() error, out chan<- *llm.Chunk) {
	send := func(c *llm.Chunk) bool {
		select {
		case out <- c:
			return true
		case <-ctx.Done():
			return false
		}
	}
	fail := func(err error) { send(&llm.Chunk{Type: llm.ChunkError, Error: err}) }

	tools := map[int32]*pendingTool{}
	var usage *llm.Usage

	for ev := range events {
		switch e := ev.(type) {
		case *types.ConverseStreamOutputMemberContentBlockStart:
			if start, ok := e.Value.Start.(*types.ContentBlockStartMemberToolUse); ok {
				tools[aws.ToInt32(e.Value.ContentBlockIndex)] = &pendingTool{
					id:   aws.ToString(start.Value.ToolUseId),
					name: aws.ToString(start.Value.Name),
				}
			}

		case *types.ConverseStreamOutputMemberContentBlockDelta:
			switch d := e.Value.Delta.(type) {
			case *types.ContentBlockDeltaMemberText:
				if d.Value != "" && !send(&llm.Chunk{Type: llm.ChunkText, Content: d.Value}) {
					return
				}
			case *types.ContentBlockDeltaMemberToolUse:
				if pt := tools[aws.ToInt32(e.Value.ContentBlockIndex)]; pt != nil {
					pt.input.WriteString(aws.ToString(d.Value.Input))
				}
			}

		case *types.ConverseStreamOutputMemberContentBlockStop:
			idx := aws.ToInt32(e.Value.ContentBlockIndex)
			pt := tools[idx]
			if pt == nil {
				continue
			}
			delete(tools, idx)
			params := pt.input.String()
			if params == "" {
				params = "{}"
			}
			if !json.Valid([]byte(params)) {
				fail(fmt.Errorf("模型返回的工具入参不是合法 JSON（工具 %s）", pt.name))
				return
			}
			if !send(&llm.Chunk{Type: llm.ChunkToolCall, ToolCall: &llm.ToolCall{ID: pt.id, Name: pt.name, Params: params}}) {
				return
			}

		case *types.ConverseStreamOutputMemberMetadata:
			if u := e.Value.Usage; u != nil {
				usage = &llm.Usage{InputTokens: int(aws.ToInt32(u.InputTokens)), OutputTokens: int(aws.ToInt32(u.OutputTokens))}
			}
		}
	}

	if err := streamErr(); err != nil {
		fail(fmt.Errorf("bedrock 流中断: %w", err))
		return
	}
	if ctx.Err() != nil {
		fail(ctx.Err())
		return
	}
	send(&llm.Chunk{Type: llm.ChunkDone, Usage: usage})
}

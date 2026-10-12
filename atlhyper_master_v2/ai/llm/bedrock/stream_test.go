package bedrock

import (
	"context"
	"errors"
	"testing"

	"AtlHyper/atlhyper_master_v2/ai/llm"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
)

// runPump 把事件序列喂给 pump，收集产出的 Chunk
func runPump(t *testing.T, events []types.ConverseStreamOutput, streamErr error) []*llm.Chunk {
	t.Helper()
	src := make(chan types.ConverseStreamOutput, len(events))
	for _, e := range events {
		src <- e
	}
	close(src)
	out := make(chan *llm.Chunk, 64)
	pump(context.Background(), src, func() error { return streamErr }, out)
	close(out)
	var chunks []*llm.Chunk
	for c := range out {
		chunks = append(chunks, c)
	}
	return chunks
}

func textDelta(idx int32, s string) types.ConverseStreamOutput {
	return &types.ConverseStreamOutputMemberContentBlockDelta{Value: types.ContentBlockDeltaEvent{
		ContentBlockIndex: aws.Int32(idx), Delta: &types.ContentBlockDeltaMemberText{Value: s},
	}}
}

func toolStart(idx int32, id, name string) types.ConverseStreamOutput {
	return &types.ConverseStreamOutputMemberContentBlockStart{Value: types.ContentBlockStartEvent{
		ContentBlockIndex: aws.Int32(idx),
		Start:             &types.ContentBlockStartMemberToolUse{Value: types.ToolUseBlockStart{ToolUseId: aws.String(id), Name: aws.String(name)}},
	}}
}

func toolDelta(idx int32, frag string) types.ConverseStreamOutput {
	return &types.ConverseStreamOutputMemberContentBlockDelta{Value: types.ContentBlockDeltaEvent{
		ContentBlockIndex: aws.Int32(idx),
		Delta:             &types.ContentBlockDeltaMemberToolUse{Value: types.ToolUseBlockDelta{Input: aws.String(frag)}},
	}}
}

func blockStop(idx int32) types.ConverseStreamOutput {
	return &types.ConverseStreamOutputMemberContentBlockStop{Value: types.ContentBlockStopEvent{ContentBlockIndex: aws.Int32(idx)}}
}

func usageEvent(in, out int32) types.ConverseStreamOutput {
	return &types.ConverseStreamOutputMemberMetadata{Value: types.ConverseStreamMetadataEvent{
		Usage: &types.TokenUsage{InputTokens: aws.Int32(in), OutputTokens: aws.Int32(out), TotalTokens: aws.Int32(in + out)},
	}}
}

func messageStop(reason types.StopReason) types.ConverseStreamOutput {
	return &types.ConverseStreamOutputMemberMessageStop{Value: types.MessageStopEvent{StopReason: reason}}
}

func TestPump_TextAndUsage(t *testing.T) {
	chunks := runPump(t, []types.ConverseStreamOutput{
		&types.ConverseStreamOutputMemberMessageStart{},
		textDelta(0, "hel"), textDelta(0, "lo"), blockStop(0),
		messageStop(types.StopReasonEndTurn), usageEvent(12, 3),
	}, nil)

	if len(chunks) != 3 || chunks[0].Content != "hel" || chunks[1].Content != "lo" {
		t.Fatalf("chunks = %+v", chunks)
	}
	last := chunks[2]
	if last.Type != llm.ChunkDone || last.Usage == nil || last.Usage.InputTokens != 12 || last.Usage.OutputTokens != 3 {
		t.Errorf("done = %+v usage=%+v", last, last.Usage)
	}
}

// 工具入参以 JSON 片段分多次到达，按 contentBlockIndex 拼接，blockStop 时发出；两个工具交错也不串
func TestPump_ToolCallsAssembledPerIndex(t *testing.T) {
	chunks := runPump(t, []types.ConverseStreamOutput{
		textDelta(0, "checking"), blockStop(0),
		toolStart(1, "t1", "get_pods"),
		toolStart(2, "t2", "get_events"),
		toolDelta(1, `{"ns":`), toolDelta(2, `{}`), toolDelta(1, `"default"}`),
		blockStop(2), blockStop(1),
		messageStop(types.StopReasonToolUse), usageEvent(1, 1),
	}, nil)

	var calls []*llm.ToolCall
	for _, c := range chunks {
		if c.Type == llm.ChunkToolCall {
			calls = append(calls, c.ToolCall)
		}
	}
	if len(calls) != 2 {
		t.Fatalf("tool calls = %+v", calls)
	}
	if calls[0].ID != "t2" || calls[0].Name != "get_events" || calls[0].Params != `{}` {
		t.Errorf("call0 = %+v", calls[0])
	}
	if calls[1].ID != "t1" || calls[1].Name != "get_pods" || calls[1].Params != `{"ns":"default"}` {
		t.Errorf("call1 = %+v", calls[1])
	}
	if chunks[len(chunks)-1].Type != llm.ChunkDone {
		t.Error("最后应为 done")
	}
}

// 没有任何入参片段的工具调用 → Params 为 {}
func TestPump_ToolWithoutInput(t *testing.T) {
	chunks := runPump(t, []types.ConverseStreamOutput{
		toolStart(0, "t1", "ping"), blockStop(0), messageStop(types.StopReasonToolUse),
	}, nil)
	if chunks[0].Type != llm.ChunkToolCall || chunks[0].ToolCall.Params != `{}` {
		t.Errorf("chunks = %+v", chunks[0])
	}
}

// 推理内容（Qwen / gpt-oss 会出）不得混进正文
func TestPump_ReasoningIgnored(t *testing.T) {
	chunks := runPump(t, []types.ConverseStreamOutput{
		&types.ConverseStreamOutputMemberContentBlockDelta{Value: types.ContentBlockDeltaEvent{
			ContentBlockIndex: aws.Int32(0),
			Delta:             &types.ContentBlockDeltaMemberReasoningContent{Value: &types.ReasoningContentBlockDeltaMemberText{Value: "let me think"}},
		}},
		textDelta(1, "answer"), messageStop(types.StopReasonEndTurn),
	}, nil)
	if len(chunks) != 2 || chunks[0].Content != "answer" {
		t.Errorf("chunks = %+v", chunks)
	}
}

// 流中途出错：发 error，不发 done（上层据此区分「出错」与「正常收尾」）
func TestPump_StreamErrorIsReported(t *testing.T) {
	chunks := runPump(t, []types.ConverseStreamOutput{textDelta(0, "par")}, errors.New("throttled"))
	last := chunks[len(chunks)-1]
	if last.Type != llm.ChunkError || last.Error == nil {
		t.Errorf("last = %+v", last)
	}
	for _, c := range chunks {
		if c.Type == llm.ChunkDone {
			t.Error("出错时不应发 done")
		}
	}
}

// 工具入参不是合法 JSON（模型输出坏掉）→ 报错，而不是把坏 JSON 交给工具执行器
func TestPump_InvalidToolInputIsError(t *testing.T) {
	chunks := runPump(t, []types.ConverseStreamOutput{
		toolStart(0, "t1", "get_pods"), toolDelta(0, `{"ns":`), blockStop(0),
	}, nil)
	last := chunks[len(chunks)-1]
	if last.Type != llm.ChunkError {
		t.Errorf("last = %+v", last)
	}
}

package llm

import (
	"context"
	"errors"
	"strings"
)

// ErrStreamIncomplete 流在没有完成标记的情况下结束（连接中断、提供商没按约定收尾）。
// 半截内容 ⛔ 不能当作完整回答入库。
var ErrStreamIncomplete = errors.New("模型的流在完成前就结束了")

// StreamResult 一轮流式回答读完后的结果
type StreamResult struct {
	Text         string
	ToolCalls    []ToolCall
	Usage        *Usage // 只有正常完成时才有（各家都在结尾给用量）
	ProviderTurn any    // 整轮回答的提供商原始形式，下一轮请求原样带上（见 Message.ProviderTurn）
}

// ReadStream 读完一轮流式回答。各家适配器的流都必须以 ChunkDone 或 ChunkError 结束：
// 通道关闭时两者都没有，视为中断（ErrStreamIncomplete）；ctx 取消返回 ctx.Err()。
// 出错时已收到的部分照样返回。onText 每收到一段文字调用一次（可为 nil）。
func ReadStream(ctx context.Context, stream <-chan *Chunk, onText func(string)) (StreamResult, error) {
	var text strings.Builder
	var result StreamResult
	finish := func(err error) (StreamResult, error) {
		result.Text = text.String()
		return result, err
	}
	for {
		select {
		case <-ctx.Done():
			return finish(ctx.Err())
		case chunk, ok := <-stream:
			if !ok {
				return finish(ErrStreamIncomplete)
			}
			switch chunk.Type {
			case ChunkText:
				text.WriteString(chunk.Content)
				if onText != nil {
					onText(chunk.Content)
				}
			case ChunkToolCall:
				if chunk.ToolCall != nil {
					result.ToolCalls = append(result.ToolCalls, *chunk.ToolCall)
				}
			case ChunkDone:
				result.Usage = chunk.Usage
				result.ProviderTurn = chunk.ProviderTurn
				return finish(nil)
			case ChunkError:
				if chunk.Error == nil {
					return finish(errors.New("LLM 流错误（提供商没有给出原因）"))
				}
				return finish(chunk.Error)
			}
		}
	}
}

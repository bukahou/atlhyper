package llm

import "context"

// SendChunk 适配层往流里发一段。调用方取消后可能已经不再读（ReadStream 直接返回），
// 所以取消时放弃、⛔ 不阻塞，缓冲区有空位时留下取消原因。返回 false 表示已取消，适配层应停止读上游并返回。
func SendChunk(ctx context.Context, ch chan<- *Chunk, chunk *Chunk) bool {
	select {
	case ch <- chunk:
		return true
	case <-ctx.Done():
		select {
		case ch <- &Chunk{Type: ChunkError, Error: ctx.Err()}:
		default:
		}
		return false
	}
}

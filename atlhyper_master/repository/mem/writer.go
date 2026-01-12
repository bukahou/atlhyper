// repository/mem/writer.go
// 内存仓库写入实现
package mem

import (
	"context"

	"AtlHyper/atlhyper_master/store/memory"
	"AtlHyper/model/transport"
)

// HubWriter 基于内存 Hub 的 Writer 实现
type HubWriter struct{}

func (HubWriter) AppendEnvelope(ctx context.Context, env transport.Envelope) error {
	memory.AppendEnvelope(env)
	return nil
}

func (HubWriter) AppendEnvelopeBatch(ctx context.Context, envs []transport.Envelope) error {
	memory.AppendEnvelopeBatch(envs)
	return nil
}

func (HubWriter) ReplaceLatest(ctx context.Context, env transport.Envelope) error {
	memory.ReplaceLatest(env)
	return nil
}

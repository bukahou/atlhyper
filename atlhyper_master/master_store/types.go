// external/master_store/types.go
package master_store

import (
	"AtlHyper/model/transport"
	"encoding/json"
	"time"
)

// EnvelopeRecord 是 Master 侧统一入池的"壳记录"。
// -----------------------------------------------------------------------------
// - 设计目的：
//     * Master 不直接存储业务结构（LogEvent / NodeMetrics / Pod 等）
//     * 而是存储统一的 EnvelopeRecord，只包含 Envelope 的元信息 + 原始 Payload
// - 优点：
//     * 避免强耦合，Store 无需理解 Payload 内部格式
//     * 读取时可以根据 Source / ClusterID / 时间窗口做统一筛选
//     * 业务方需要具体数据时，再在 Payload 上解码成自己的结构
// -----------------------------------------------------------------------------
// 字段说明：
//   Version    → 协议版本（例如 v1）
//   ClusterID  → 集群 ID（一般是 kube-system namespace 的 UID）
//   Source     → 数据源标识（如 "k8s_event" / "metrics_agent"）
//   SentAtMs   → Agent 发送时刻（由 Envelope.ts_ms 提供，毫秒级时间戳）
//   EnqueuedAt → Master 入池时刻（本地接收的时间，用于 TTL 清理等）
//   Payload    → 原始载荷（json.RawMessage，保持原样存储，不解析）
// -----------------------------------------------------------------------------
type EnvelopeRecord struct {
	Version     string          // 协议版本（例如 v1）
	ClusterID   string          // 集群ID（kube-system ns 的 UID）
	Source      string          // 数据源标识（k8s_event / metrics_agent / ...）
	SentAtMs    int64           // Agent 发送时刻（毫秒时间戳，来自 Envelope.ts_ms）
	EnqueuedAt  time.Time       // Master 入池时间（接收落入 Store 的时刻）
	Payload     json.RawMessage // 原始载荷（保持原封不动）
}

// NewRecordFromEnvelope：直接使用参数构造 EnvelopeRecord。
// -----------------------------------------------------------------------------
// - 使用场景：若你手头有分散的字段（ver/clusterID/...），
//   可直接调用本函数快速生成 EnvelopeRecord。
// - 注意：
//     * EnqueuedAt 使用当前时间
//     * SentAtMs 使用上报包自带的 ts_ms
// -----------------------------------------------------------------------------
func NewRecordFromEnvelope(ver, clusterID, source string, tsMs int64, payload json.RawMessage) EnvelopeRecord {
	return EnvelopeRecord{
		Version:    ver,
		ClusterID:  clusterID,
		Source:     source,
		SentAtMs:   tsMs,
		EnqueuedAt: time.Now(),
		Payload:    payload,
	}
}

func NewRecordFromStd(env transport.Envelope) EnvelopeRecord {
    return EnvelopeRecord{
        Version:    env.Version,
        ClusterID:  env.ClusterID,
        Source:     env.Source,
        SentAtMs:   env.TimestampMs,
        EnqueuedAt: time.Now(),
        Payload:    env.Payload,
    }
}


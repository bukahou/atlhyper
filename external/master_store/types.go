// external/master_store/types.go
package master_store

import (
	"NeuroController/model/envelope"
	"encoding/json"
	"time"
)

// EnvelopeRecord 是 Master 侧统一入池的“壳记录”。
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

func NewRecordFromStd(env envelope.Envelope) EnvelopeRecord {
    return EnvelopeRecord{
        Version:    env.Version,
        ClusterID:  env.ClusterID,
        Source:     env.Source,
        SentAtMs:   env.TimestampMs,
        EnqueuedAt: time.Now(),
        Payload:    env.Payload,
    }
}






// MinimalEnvelope 接口：最小化抽象，便于与不同 Envelope 定义对接。
// -----------------------------------------------------------------------------
// - 设计目的：
//     * 避免 master_store 强依赖某一个 Envelope 实现（如 utils.Envelope）
//     * 任何实现了 GetVersion/GetClusterID/... 方法的类型都能转换成 EnvelopeRecord
// - 使用场景：
//     * 在 receivers、agent 或其他模块里，只要能提供这些 getter 方法，
//       就可以直接调用 NewRecordFrom(minimalEnvelope) 写入 Store。
// -----------------------------------------------------------------------------
// type MinimalEnvelope interface {
// 	GetVersion() string
// 	GetClusterID() string
// 	GetSource() string
// 	GetTimestampMs() int64
// 	GetPayload() json.RawMessage
// }

// NewRecordFrom：从 MinimalEnvelope 接口构造 EnvelopeRecord。
// -----------------------------------------------------------------------------
// - 使用场景：接收器 Handler 若绑定的是实现了 MinimalEnvelope 的类型，
//   可直接调用本函数生成 EnvelopeRecord。
// - 注意：内部同样会设置 EnqueuedAt=当前时间。
// -----------------------------------------------------------------------------
// func NewRecordFrom(min MinimalEnvelope) EnvelopeRecord {
// 	return EnvelopeRecord{
// 		Version:    min.GetVersion(),
// 		ClusterID:  min.GetClusterID(),
// 		Source:     min.GetSource(),
// 		SentAtMs:   min.GetTimestampMs(),
// 		EnqueuedAt: time.Now(),
// 		Payload:    min.GetPayload(),
// 	}
// }

// NewRecordFromStd：从标准 envelope.Envelope 转换为 EnvelopeRecord。
// -----------------------------------------------------------------------------
// - 使用场景：如果你接收的是标准定义的 envelope.Envelope（推荐统一使用），
//   可直接调用本函数转换成 EnvelopeRecord。
// - 注意：
//     * EnqueuedAt 仍然使用 Master 当前时间
//     * SentAtMs 来自 Envelope.TimestampMs
// -----------------------------------------------------------------------------

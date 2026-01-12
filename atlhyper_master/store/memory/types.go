// store/memory/types.go
// 内存存储类型定义
package memory

import (
	"AtlHyper/model/transport"
	"encoding/json"
	"time"
)

// EnvelopeRecord 是 Master 侧统一入池的"壳记录"
// 设计目的：Master 不直接存储业务结构，而是存储统一的 EnvelopeRecord
type EnvelopeRecord struct {
	Version    string          // 协议版本
	ClusterID  string          // 集群ID
	Source     string          // 数据源标识
	SentAtMs   int64           // Agent 发送时刻（毫秒时间戳）
	EnqueuedAt time.Time       // Master 入池时间
	Payload    json.RawMessage // 原始载荷
}

// NewRecordFromEnvelope 从字段构造 EnvelopeRecord
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

// NewRecordFromStd 从标准 Envelope 构造
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

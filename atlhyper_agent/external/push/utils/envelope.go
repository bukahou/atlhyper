// internal/push/utils/envelope.go
package utils

import (
	"AtlHyper/model/envelope"
	"encoding/json"
	"time"
)

// EnvelopeVersion 是当前上报包裹的版本号。
// 保留为常量，后续协议演进时可新增 v2/v3 并做并行兼容。
const EnvelopeVersion = "v1"

// NewEnvelope 便捷构造函数：自动填充 Version 与当前毫秒时间戳。
// 其余字段由调用者显式传入，保持可读性与可控性。
func NewEnvelope(clusterID, source string, payload json.RawMessage) envelope.Envelope {
	return envelope.Envelope{
		Version:     EnvelopeVersion,
		ClusterID:   clusterID,
		Source:      source,
		Payload:     payload,
		TimestampMs: time.Now().UnixMilli(),
	}
}

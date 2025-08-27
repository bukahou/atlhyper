// internal/push/utils/envelope.go
package utils

import (
	"NeuroController/model/envelope"
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


// Envelope 定义了推送数据的通用外壳。
// 各类推送器（events/metrics/heartbeat）将自身业务数据序列化后
// 放入 Payload 字段，Manager/调度器与 HTTP 客户端都只感知 Envelope。
// type Envelope struct {
// 	Version     string          `json:"version"`   // 协议版本：固定 "v1"
// 	ClusterID   string          `json:"cluster_id"` // kube-system namespace 的 UID
// 	Source      string          `json:"source"`    // 数据源标识：如 "k8s_event"
// 	TimestampMs int64           `json:"ts_ms"`     // 发送时刻（毫秒）
// 	Payload     json.RawMessage `json:"payload"`   // 业务负载（各 pusher 自行定义结构）
// }

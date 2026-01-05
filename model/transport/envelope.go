// model/transport/envelope.go
// 传输封装协议（Agent ↔ Master）
package transport

import "encoding/json"

type Envelope struct {
	Version     string          `json:"version"`
	ClusterID   string          `json:"cluster_id"`
	Source      string          `json:"source"`
	TimestampMs int64           `json:"ts_ms"`
	Payload     json.RawMessage `json:"payload"`
}

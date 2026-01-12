// logic/pusher/envelope.go
// Envelope 数据封装
package pusher

import (
	"encoding/json"
	"time"

	"AtlHyper/model/transport"
)

const EnvelopeVersion = "v1"

// NewEnvelope 创建数据信封
func NewEnvelope(clusterID, source string, payload json.RawMessage) transport.Envelope {
	return transport.Envelope{
		Version:     EnvelopeVersion,
		ClusterID:   clusterID,
		Source:      source,
		Payload:     payload,
		TimestampMs: time.Now().UnixMilli(),
	}
}

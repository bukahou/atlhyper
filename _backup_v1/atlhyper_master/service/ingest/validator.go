// service/ingest/validator.go
package ingest

import (
	"errors"

	"AtlHyper/model/transport"
)

// Validator 数据校验器
// -----------------------------------------------------------------------------
// 职责：
//   - 校验 Envelope 必填字段
//   - 校验 Source 类型是否匹配
//   - 可扩展：Payload 格式校验、数据完整性校验等
// -----------------------------------------------------------------------------
type Validator struct{}

// NewValidator 创建校验器实例
func NewValidator() *Validator {
	return &Validator{}
}

// 校验错误定义
var (
	ErrEmptyVersion   = errors.New("envelope version is empty")
	ErrEmptyClusterID = errors.New("envelope cluster_id is empty")
	ErrEmptySource    = errors.New("envelope source is empty")
	ErrInvalidSource  = errors.New("envelope source mismatch")
	ErrInvalidTimestamp = errors.New("envelope timestamp_ms is invalid")
	ErrEmptyPayload   = errors.New("envelope payload is empty")
)

// ValidateEnvelope 校验 Envelope 基本字段
// -----------------------------------------------------------------------------
// 参数：
//   - env: 待校验的 Envelope
//   - expectedSource: 期望的 Source 类型（如 pod_list_snapshot）
// 返回：
//   - error: 校验失败时返回具体错误
// -----------------------------------------------------------------------------
func (v *Validator) ValidateEnvelope(env transport.Envelope, expectedSource string) error {
	// 1. 校验 Version
	if env.Version == "" {
		return ErrEmptyVersion
	}

	// 2. 校验 ClusterID
	if env.ClusterID == "" {
		return ErrEmptyClusterID
	}

	// 3. 校验 Source
	if env.Source == "" {
		return ErrEmptySource
	}
	if env.Source != expectedSource {
		return ErrInvalidSource
	}

	// 4. 校验 Timestamp
	if env.TimestampMs <= 0 {
		return ErrInvalidTimestamp
	}

	// 5. 校验 Payload
	if len(env.Payload) == 0 {
		return ErrEmptyPayload
	}

	return nil
}

// ValidateEnvelopeBasic 仅校验基本字段（不校验 Source 类型）
// -----------------------------------------------------------------------------
// 适用场景：通用校验，不关心具体 Source 类型
// -----------------------------------------------------------------------------
func (v *Validator) ValidateEnvelopeBasic(env transport.Envelope) error {
	if env.Version == "" {
		return ErrEmptyVersion
	}
	if env.ClusterID == "" {
		return ErrEmptyClusterID
	}
	if env.Source == "" {
		return ErrEmptySource
	}
	if env.TimestampMs <= 0 {
		return ErrInvalidTimestamp
	}
	if len(env.Payload) == 0 {
		return ErrEmptyPayload
	}
	return nil
}

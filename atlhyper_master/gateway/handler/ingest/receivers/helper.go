// gateway/handler/ingest/receivers/helper.go
package receivers

import (
	"encoding/json"
	"io"
	"net/http"

	ziputil "AtlHyper/common"
	"AtlHyper/model/transport"

	"github.com/gin-gonic/gin"
)

// ParseEnvelopeConfig 解析配置
type ParseEnvelopeConfig struct {
	MaxCompressedSize   int64 // 压缩前最大字节数
	MaxDecompressedSize int64 // 解压后最大字节数
}

// DefaultParseConfig 默认解析配置
var DefaultParseConfig = ParseEnvelopeConfig{
	MaxCompressedSize:   16 << 20, // 16MiB
	MaxDecompressedSize: 64 << 20, // 64MiB
}

// SmallParseConfig 小数据解析配置（用于事件等）
var SmallParseConfig = ParseEnvelopeConfig{
	MaxCompressedSize:   1 << 20, // 1MiB
	MaxDecompressedSize: 8 << 20, // 8MiB
}

// MediumParseConfig 中等数据解析配置（用于指标等）
var MediumParseConfig = ParseEnvelopeConfig{
	MaxCompressedSize:   2 << 20,  // 2MiB
	MaxDecompressedSize: 16 << 20, // 16MiB
}

// ParseEnvelope 解析请求体为 Envelope
// -----------------------------------------------------------------------------
// 功能：
//   - 限制请求体大小
//   - 自动解压 gzip
//   - 解析 JSON 为 Envelope
// 返回：
//   - *transport.Envelope: 解析成功时返回 Envelope 指针
//   - bool: 解析是否成功（失败时已写入响应）
// -----------------------------------------------------------------------------
func ParseEnvelope(c *gin.Context, cfg ParseEnvelopeConfig) (*transport.Envelope, bool) {
	// 1) 限制压缩前的请求体大小
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, cfg.MaxCompressedSize)

	// 2) 自动解压或透传
	rc, err := ziputil.MaybeGunzipReaderAuto(c.Request.Body, c.GetHeader("Content-Encoding"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "解压失败或请求体非法"})
		return nil, false
	}
	defer rc.Close()

	// 3) 限制解压后的可读字节
	rcLimited := io.LimitReader(rc, cfg.MaxDecompressedSize)

	// 4) 解析 JSON Envelope
	var env transport.Envelope
	if err := json.NewDecoder(rcLimited).Decode(&env); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "请求体解析失败：不是有效的 JSON Envelope"})
		return nil, false
	}

	return &env, true
}

// RespondValidationError 响应校验错误
func RespondValidationError(c *gin.Context, err error) {
	c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
}

// RespondServiceError 响应服务错误
func RespondServiceError(c *gin.Context, err error) {
	c.JSON(http.StatusInternalServerError, gin.H{"message": "处理失败: " + err.Error()})
}

// RespondSuccess 响应成功（204 No Content）
func RespondSuccess(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

package receivers

import (
	"encoding/json"
	"io"
	"net/http"

	"AtlHyper/atlhyper_master/master_store"
	"AtlHyper/model/transport"
	ziputil "AtlHyper/common" // 根目录通用 gzip 工具

	"github.com/gin-gonic/gin"
)

// 仅接收 pod 列表快照
const SourcePodListSnapshot = transport.SourcePodListSnapshot

// HandlePodListSnapshotIngest 处理 /engest/pods/snapshot
// - 兼容压缩与未压缩：支持 Content-Encoding:gzip，且自动嗅探魔数
// - 校验必要字段：Version / ClusterID / TimestampMs / Payload
// - 成功返回 204，无响应体
func HandlePodListSnapshotIngest(c *gin.Context) {
	// 1) 限制“压缩前”的请求体大小，拦截异常大包
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 16<<20) // 16MiB

	// 2) 自动解压或透传（头部 + 魔数嗅探），拿到一个可读取解压后数据的 Reader
	rc, err := ziputil.MaybeGunzipReaderAuto(c.Request.Body, c.GetHeader("Content-Encoding"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "解压失败或请求体非法"})
		return
	}
	defer rc.Close()

	// （可选）3) 再次限制“解压后”的最大可读字节，防 zip-bomb 放大
	// 按需调整上限，例如 64MiB
	rcLimited := io.LimitReader(rc, 64<<20)

	// 4) 解析 JSON Envelope
	var env transport.Envelope
	if err := json.NewDecoder(rcLimited).Decode(&env); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "请求体解析失败：不是有效的 JSON Envelope"})
		return
	}

	// 5) 基本校验
	if env.Source != SourcePodListSnapshot {
		c.JSON(http.StatusBadRequest, gin.H{"message": "source 非法：仅支持 pod_list_snapshot"})
		return
	}
	if env.Version == "" || env.ClusterID == "" || env.TimestampMs <= 0 || len(env.Payload) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Envelope 字段缺失：version/cluster_id/ts_ms/payload 均为必填"})
		return
	}

	// 6) 入池（Master 全局内存池；后续由消费者解码处理）
	master_store.ReplaceLatest(env)

	// 7) 成功：204 无 body
	c.Status(http.StatusNoContent)
}

package control

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"time"

	ziputil "AtlHyper/utils"

	"github.com/gin-gonic/gin"
)

// HandleWatch —— Agent 长轮询接口
// -----------------------------------------------------------------------------
// - 调用方：Agent
// - 功能：获取最新的 CommandSet（命令副本）
// - 流程：
//   1. 解析请求参数：clusterID / rv（已知版本号）/ waitSeconds（最长等待秒数）
//   2. 如果当前有新版本（rv 落后），立即返回最新 CommandSet
//   3. 如果没有新版本，则挂起等待 waitSeconds；
//      - 若期间有更新 → 返回最新 CommandSet
//      - 若超时仍无更新 → 返回 304 Not Modified
// - 特点：模仿 K8s List+Watch 模型，保证 Agent 与 Master 副本一致
// HandleWatch —— Agent 长轮询接口（兼容 gzip）

func HandleWatch(c *gin.Context) {
    // 限制压缩前大小
    c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 4<<20) // 4MiB 足够

    // 自动解压（按 Content-Encoding 和魔数嗅探）
    rc, err := ziputil.MaybeGunzipReaderAuto(c.Request.Body, c.GetHeader("Content-Encoding"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error":"invalid or gzipped body decompress failed"})
        return
    }
    defer rc.Close()

    // 限制解压后大小
    r := io.LimitReader(rc, 8<<20)

    // 手动解析 JSON
    var req struct {
        ClusterID   string `json:"clusterID"`
        RV          string `json:"rv"`
        WaitSeconds int    `json:"waitSeconds"`
    }
    if err := json.NewDecoder(r).Decode(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error":"bad json"})
        return
    }
    if req.WaitSeconds <= 0 { req.WaitSeconds = 30 }

    var rv uint64
    if req.RV != "" {
        rv, _ = strconv.ParseUint(req.RV, 10, 64)
    }

    // 先看是否已更新
    if set, ok := getIfNewer(req.ClusterID, rv); ok {
        c.JSON(http.StatusOK, set)
        return
    }

    // 长轮询等待
    if set, ok := waitChange(req.ClusterID, time.Duration(req.WaitSeconds)*time.Second); ok {
        c.JSON(http.StatusOK, set)
    } else {
        c.Status(http.StatusNotModified)
    }
}


// HandleAck —— Agent 执行结果回执接口
// -----------------------------------------------------------------------------
// - 调用方：Agent
// - 功能：在执行完命令后，把结果（AckResult）回报给 Master
// - 流程：
//   1. 解析请求参数：clusterID + results（执行结果列表）
//   2. 调用 applyAck 更新 CommandSet（清理成功的命令 / 标记失败的命令）
//   3. 返回 {"ok":true}
// - 特点：建立起“下发 → 执行 → 回执”的闭环，便于 Master 审计与后续处理
// HandleAck —— Agent 执行结果回执接口（兼容 gzip）
func HandleAck(c *gin.Context) {
    c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 4<<20)

    rc, err := ziputil.MaybeGunzipReaderAuto(c.Request.Body, c.GetHeader("Content-Encoding"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error":"invalid or gzipped body decompress failed"})
        return
    }
    defer rc.Close()

    r := io.LimitReader(rc, 8<<20)

    var req struct {
        ClusterID string      `json:"clusterID"`
        Results   []AckResult `json:"results"`
    }
    if err := json.NewDecoder(r).Decode(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error":"bad json"})
        return
    }

    applyAck(req.ClusterID, req.Results)
    c.JSON(http.StatusOK, gin.H{"ok": true})
}


// HandleEnqueue —— 管理端/测试命令入队接口
// -----------------------------------------------------------------------------
// - 调用方：Master 管理界面 / 测试工具
// - 功能：往某个集群的 CommandSet 中追加一条新命令
// - 流程：
//   1. 解析请求参数：clusterID + command（命令详情）
//   2. 调用 upsertCommand，把命令放进副本，并触发 RV++ 唤醒等待的 Agent
//   3. 返回 {"ok":true}
// - 特点：是 Master 主动下发命令的入口，Agent 在下一次 watch 时会拿到该命令
func HandleEnqueue(c *gin.Context) {
    var req struct {
        ClusterID string  `json:"clusterID"`
        Command   Command `json:"command"`
    }
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    upsertCommand(req.ClusterID, req.Command)
    c.JSON(http.StatusOK, gin.H{"ok": true})
}

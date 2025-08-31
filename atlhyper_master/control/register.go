package control

import "github.com/gin-gonic/gin"

// RegisterControlRoutes —— 注册控制通道（ops）的所有路由
// -----------------------------------------------------------------------------
// 路由前缀：/ingest/ops
// 说明：这些接口属于 Master → Agent 的“操作下发 / 回执”通道，
//       与普通的资源上报（/ingest/...list）区分开。
//       目前定义了三类接口：Watch、Ack、Enqueue。
// -----------------------------------------------------------------------------
// 1) POST /ingest/ops/watch
//    - 用途：Agent 发起长轮询（List+Watch 模式），请求 Master 返回最新命令副本。
//    - 入参：clusterID（Header X-Cluster-ID）+ 请求体 { rv, waitSeconds }
//    - 返回：CommandSet（如果 rv 落后或有新命令）；若无更新则 304 或空。
//    - 特点：Agent 会一直循环调用，保持和 Master 的副本同步。
//
// 2) POST /ingest/ops/ack
//    - 用途：Agent 在执行完命令后，把执行结果回执给 Master。
//    - 入参：clusterID（Header）+ 请求体 { results: []AckResult }
//    - 返回：{"ok":true} 或 {"code":20000}
//    - 特点：Master 更新副本（如清理成功的命令），并触发 rv++，便于后续 Watch 看到变化。
//
// 3) POST /ingest/ops/enqueue
//    - 用途：管理端 / 测试用，把一条命令塞到某个集群的副本中。
//    - 入参：clusterID + Command（命令详情）
//    - 返回：{"ok":true}
//    - 特点：用于模拟/下发控制任务（如 Pod 重启、镜像更新等）；
//            在生产场景可对接 UI 或外部 API，由运维/控制器触发。
// -----------------------------------------------------------------------------
func RegisterControlRoutes(rg *gin.RouterGroup) {
    ctrl := rg.Group("/ops")
    ctrl.POST("/watch", HandleWatch)   // Agent → Master：长轮询获取命令副本
    ctrl.POST("/ack", HandleAck)       // Agent → Master：命令执行回执
    ctrl.POST("/enqueue", HandleEnqueue) // Master 管理端：手工/测试下发命令
}

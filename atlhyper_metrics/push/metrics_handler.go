package push

import (
	"AtlHyper/atlhyper_metrics/internal"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
)

// StartPushServer 启动独立的 Metrics 采集进程的 HTTP 服务。
// 注意：这个服务属于“采集端 (collector)”，并不是 Agent。
// 职责：
// 1) 提供只读 REST 接口（/metrics/snapshot），用于人工或 UI 调试拉取当前快照；
// 2) 同时在后台按固定间隔把采集到的快照主动 Push 到“Agent 接收端”（由环境变量配置）。
//
// 环境变量（由 push/reporter.go 读取）:
// - PUSH_ENABLE=true      // 是否启用主动上报（默认不启用）
// - PUSH_URL=https://...  // Agent 接收端地址（必填，启用上报时）
// - PUSH_TOKEN=xxx        // 可选：HTTP Bearer Token
// - PUSH_INTERVAL=5s      // 上报间隔，默认 5s
// - PUSH_TIMEOUT=5s       // 上报超时，默认 5s
func StartPushServer(port int) {
    // 1) 以 Release 模式启动一个轻量 HTTP 服务，仅用于调试/可视化拉取
    gin.SetMode(gin.ReleaseMode)
    router := gin.Default()

    // 2) 注册只读 REST 路由：/metrics/snapshot
    //    —— 不依赖 Agent，直接聚合本地采集数据返回，方便测试。
    api := router.Group("/metrics")
    RegisterUIAPIRoutes(api)

    // 3) 启动“主动上报”后台任务：
    //    —— 采集端会根据环境变量，定期把快照 POST 到 Agent 接收端。
    ctx, cancel := context.WithCancel(context.Background())
    StartReporterFromEnv(ctx)

    // 4) 优雅退出：收到 SIGINT/SIGTERM 时取消上报 goroutine，再退出进程
    go func() {
        ch := make(chan os.Signal, 1)
        signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
        <-ch
        cancel()
    }()

    // 5) 监听端口（仅供本地/调试调用；生产环境可选不开此端口）
    log.Printf("🚀 [Collector] Metrics HTTP 服务启动（仅调试用），监听端口: %d", port)
    if err := router.Run(fmt.Sprintf(":%d", port)); err != nil {
        log.Fatalf("❌ Collector HTTP 启动失败: %v", err)
    }
}

// RegisterUIAPIRoutes 注册只读的调试接口。
// 注意：这些接口返回的是“采集端”视角的实时快照，
// 不依赖 Agent，因此可以在不连 Agent 的情况下自检。
func RegisterUIAPIRoutes(router *gin.RouterGroup) {
    // GET /metrics/snapshot
    // 用于调试/自检：聚合 CPU、内存、磁盘、网络、温度、Top 等数据并返回。
    router.GET("/snapshot", HandleGetNodeMetricsSnapshot)
}

// HandleGetNodeMetricsSnapshot 返回采集端的当前快照。
// 该聚合直接读取采集模块的缓存（如 CPU 为后台采样缓存），
// 与 Push 上报使用同一套数据来源，保证调试与上报一致性。
func HandleGetNodeMetricsSnapshot(c *gin.Context) {
    snapshot := internal.BuildNodeMetricsSnapshot()
    c.JSON(http.StatusOK, snapshot)
}

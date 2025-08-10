package server

import (
	"github.com/gin-gonic/gin"

	"NeuroController/internal/ingest/receivers"
	"NeuroController/internal/ingest/store"
)

// RegisterIngestRoutes 将所有 ingest 相关的路由注册到给定的路由组 g。
// - g：建议传入 /ingest 前缀，如 engine.Group("/ingest")
// - st：内存存储（metrics_store）
// - maxBodyBytes：请求体大小限制；<=0 时使用默认 2MiB
func RegisterIngestRoutes(g *gin.RouterGroup, st *store.Store, maxBodyBytes int64) {
    // 1) Metrics 接收路由组
    metricsGroup := g.Group("/metrics")
    receivers.RegisterMetricsRoutes(metricsGroup, st, maxBodyBytes)


}

    // 2) eBPF 接收路由组（预留）
    // ebpfGroup := g.Group("/ebpf")
    // receivers.RegisterEBPFRoutes(ebpfGroup, st, maxBodyBytes)

    // 3) Traces 接收路由组（预留）
    // tracesGroup := g.Group("/traces")
    // receivers.RegisterTracesRoutes(tracesGroup, st, maxBodyBytes)

    // 4) Logs 接收路由组（预留）
    // logsGroup := g.Group("/logs")
    // receivers.RegisterLogsRoutes(logsGroup, st, maxBodyBytes)
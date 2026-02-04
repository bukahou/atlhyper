// cmd/atlhyper_metrics_v2/main.go
// Metrics V2 启动入口
package main

import (
	"context"
	"os"

	"AtlHyper/atlhyper_metrics_v2"
	"AtlHyper/atlhyper_metrics_v2/config"
	"AtlHyper/common/logger"
)

var log = logger.Module("Metrics")

func main() {
	// 加载配置
	config.Load()

	// 初始化日志
	logger.Init(logger.Config{
		Level:  config.GlobalConfig.Log.Level,
		Format: config.GlobalConfig.Log.Format,
	})

	log.Info("AtlHyper Metrics V2 starting...")

	// 创建 Metrics 实例
	metrics := atlhyper_metrics_v2.New()

	// 运行 Metrics
	if err := metrics.Run(context.Background()); err != nil {
		log.Error("Metrics 运行错误", "err", err)
		os.Exit(1)
	}
}

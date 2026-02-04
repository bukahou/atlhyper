// cmd/atlhyper_master_v2/main.go
// Master V2 启动入口
package main

import (
	"context"
	"os"

	"AtlHyper/atlhyper_master_v2"
	"AtlHyper/atlhyper_master_v2/config"
	"AtlHyper/common/logger"
)

var log = logger.Module("Master")

func main() {
	// 加载配置（优先，日志配置也在其中）
	config.LoadConfig()

	// 初始化日志（使用配置中的设置）
	logger.Init(logger.Config{
		Level:  config.GlobalConfig.Log.Level,
		Format: config.GlobalConfig.Log.Format,
	})

	log.Info("AtlHyper Master V2 starting...")

	// 创建 Master 实例
	master, err := atlhyper_master_v2.New()
	if err != nil {
		log.Error("创建 Master 失败", "err", err)
		os.Exit(1)
	}

	// 运行 Master
	if err := master.Run(context.Background()); err != nil {
		log.Error("Master 运行错误", "err", err)
		os.Exit(1)
	}
}

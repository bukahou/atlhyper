// cmd/atlhyper_master_v2/main.go
// Master V2 启动入口
package main

import (
	"context"
	"log"

	"AtlHyper/atlhyper_master_v2"
	"AtlHyper/atlhyper_master_v2/config"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("AtlHyper Master V2 starting...")

	// 加载配置
	config.LoadConfig()

	// 创建 Master 实例
	master, err := atlhyper_master_v2.New()
	if err != nil {
		log.Fatalf("Failed to create master: %v", err)
	}

	// 运行 Master
	if err := master.Run(context.Background()); err != nil {
		log.Fatalf("Master error: %v", err)
	}
}

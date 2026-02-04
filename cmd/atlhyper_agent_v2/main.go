// AtlHyper Agent V2 入口程序
//
// Agent V2 是多集群 K8s 监控系统的数据采集组件，部署在每个被监控的 K8s 集群中。
//
// 主要功能:
//   - 定时采集集群资源快照 (Pod, Node, Deployment 等)，推送给 Master
//   - 长轮询获取 Master 下发的指令 (扩缩容、重启、删除等)，执行后上报结果
//   - 心跳保活
//
// 架构:
//
//	┌─────────────┐
//	│   Master    │  ← 接收快照，下发指令
//	└──────┬──────┘
//	       │ HTTP (Gzip)
//	┌──────┴──────┐
//	│  Agent V2   │  ← 本程序
//	└──────┬──────┘
//	       │ client-go
//	┌──────┴──────┐
//	│  K8s API    │
//	└─────────────┘
//
// 环境变量 (见 config/defaults.go):
//   - AGENT_CLUSTER_ID: 集群标识
//   - AGENT_MASTER_URL: Master 服务地址
//   - AGENT_KUBECONFIG: kubeconfig 路径，空则使用 InCluster 模式
//   - AGENT_SNAPSHOT_INTERVAL: 快照间隔，默认 30s
//   - AGENT_HEARTBEAT_INTERVAL: 心跳间隔，默认 15s
//   - AGENT_LOG_LEVEL: 日志级别 (debug/info/warn/error)，默认 info
//   - AGENT_LOG_FORMAT: 日志格式 (text/json)，默认 text
//
// 启动示例:
//
//	AGENT_CLUSTER_ID=prod-cluster AGENT_MASTER_URL=http://master:8080 ./atlhyper_agent_v2
package main

import (
	"context"
	"os"

	agent "AtlHyper/atlhyper_agent_v2"
	"AtlHyper/atlhyper_agent_v2/config"
	"AtlHyper/common/logger"
)

var log = logger.Module("Agent")

func main() {
	// 从环境变量加载配置（优先，日志配置也在其中）
	config.LoadConfig()

	// 初始化日志（使用配置中的设置）
	logger.Init(logger.Config{
		Level:  config.GlobalConfig.Log.Level,
		Format: config.GlobalConfig.Log.Format,
	})

	log.Info("Starting AtlHyper Agent V2...")

	// 创建 Agent 实例 (内部完成所有依赖注入)
	a, err := agent.New()
	if err != nil {
		log.Error("初始化失败", "err", err)
		os.Exit(1)
	}

	// 运行 Agent (阻塞直到收到 SIGINT/SIGTERM)
	if err := a.Run(context.Background()); err != nil {
		log.Error("运行错误", "err", err)
		os.Exit(1)
	}

	log.Info("Agent 已停止")
}

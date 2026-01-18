// Package atlhyper_master_v2 Master V2 核心包
//
// 本包提供 Master 的启动器，负责:
//   - 初始化所有依赖 (DataHub, Database, Processor, Query, CommandService, AgentSDK, Gateway)
//   - 依赖注入和组装
//   - 生命周期管理 (启动、运行、停止)
//
// 架构设计:
//   - 外部访问: Gateway → Query（读取）/ CommandService（写入）→ DataHub
//   - 内部处理: AgentSDK → Processor → DataHub
//   - Gateway 禁止直接访问 DataHub
//
// 使用方式:
//
//	config.LoadConfig()
//	master, err := atlhyper_master_v2.New()
//	master.Run(ctx)
package atlhyper_master_v2

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"AtlHyper/atlhyper_master_v2/agentsdk"
	"AtlHyper/atlhyper_master_v2/config"
	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/database/sqlite"
	"AtlHyper/atlhyper_master_v2/datahub"
	"AtlHyper/atlhyper_master_v2/datahub/memory"
	"AtlHyper/atlhyper_master_v2/gateway"
	"AtlHyper/atlhyper_master_v2/processor"
	"AtlHyper/atlhyper_master_v2/query"
	"AtlHyper/atlhyper_master_v2/service"
)

// Master 是 Master V2 的主结构体
type Master struct {
	datahub        datahub.DataHub
	database       database.Database
	processor      processor.Processor
	query          query.Query
	commandService service.CommandService
	agentSDK       *agentsdk.Server
	gateway        *gateway.Server
	eventPersist   *service.EventPersistService
}

// New 创建并初始化 Master 实例
//
// 使用 config.GlobalConfig 中的配置初始化各层组件。
// 调用前必须先调用 config.LoadConfig()。
//
// 初始化顺序:
//  1. DataHub - 实时数据中心
//  2. Database - 持久化数据库
//  3. EventPersistService - Event 持久化服务
//  4. Processor - 数据处理层（写入 DataHub）
//  5. Query - 查询抽象层（读取 DataHub）
//  6. CommandService - 指令写入服务
//  7. AgentSDK - Agent 通信层
//  8. Gateway - Web API 网关
//
// 返回:
//   - *Master: Master 实例
//   - error: 初始化错误
func New() (*Master, error) {
	cfg := &config.GlobalConfig

	// 1. 初始化 DataHub
	var hub datahub.DataHub
	switch cfg.DataHub.Type {
	case "memory":
		hub = memory.New(cfg.DataHub.EventRetention, cfg.DataHub.HeartbeatExpire)
	default:
		return nil, fmt.Errorf("unsupported datahub type: %s", cfg.DataHub.Type)
	}
	log.Printf("[Master] DataHub 初始化完成: type=%s", cfg.DataHub.Type)

	// 2. 初始化 Database
	var db database.Database
	var err error
	switch cfg.Database.Type {
	case "sqlite":
		db, err = sqlite.New(cfg.Database.Path)
		if err != nil {
			return nil, fmt.Errorf("failed to init sqlite: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported database type: %s", cfg.Database.Type)
	}
	log.Printf("[Master] 数据库初始化完成: type=%s", cfg.Database.Type)

	// 执行数据库迁移
	if err := db.Migrate(); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	// 3. 初始化 EventPersistService
	eventPersist := service.NewEventPersistService(
		hub,
		db.ClusterEventRepository(),
		service.EventPersistConfig{
			RetentionDays:   cfg.Event.RetentionDays,
			MaxCount:        cfg.Event.MaxCount,
			CleanupInterval: cfg.Event.CleanupInterval,
		},
	)
	log.Println("[Master] 事件持久化服务初始化完成")

	// 4. 初始化 Processor（写入路径）
	proc := processor.New(processor.Config{
		DataHub: hub,
		OnSnapshotReceived: func(clusterID string) {
			// 触发 Event 持久化
			if err := eventPersist.Sync(clusterID); err != nil {
				log.Printf("[Master] 事件同步失败: 集群=%s, 错误=%v", clusterID, err)
			}
		},
	})
	log.Println("[Master] 数据处理器初始化完成")

	// 5. 初始化 Query（读取路径）
	q := query.NewWithEventRepo(hub, db.ClusterEventRepository())
	log.Println("[Master] 查询层初始化完成")

	// 6. 初始化 CommandService（指令写入）
	cmdService := service.NewCommandService(hub)
	log.Println("[Master] 指令服务初始化完成")

	// 7. 初始化 AgentSDK（使用 Processor）
	agentServer := agentsdk.NewServer(agentsdk.Config{
		Port:           cfg.Server.AgentSDKPort,
		CommandTimeout: cfg.Timeout.CommandPoll,
		DataHub:        hub, // 用于指令队列
		Processor:      proc,
	})
	log.Printf("[Master] AgentSDK 初始化完成: 端口=%d", cfg.Server.AgentSDKPort)

	// 8. 初始化 Gateway（使用 Query + CommandService + DataHub）
	gw := gateway.NewServer(gateway.Config{
		Port:           cfg.Server.GatewayPort,
		Query:          q,
		CommandService: cmdService,
		Database:       db,
		DataHub:        hub,
	})
	log.Printf("[Master] Gateway 初始化完成: 端口=%d", cfg.Server.GatewayPort)

	return &Master{
		datahub:        hub,
		database:       db,
		processor:      proc,
		query:          q,
		commandService: cmdService,
		agentSDK:       agentServer,
		gateway:        gw,
		eventPersist:   eventPersist,
	}, nil
}

// Run 运行 Master
//
// 启动所有组件后阻塞等待退出信号 (SIGINT/SIGTERM)。
// 收到信号后优雅停止所有组件。
//
// 参数:
//   - ctx: 上下文，可用于外部取消
//
// 返回:
//   - error: 停止时的错误
func (m *Master) Run(ctx context.Context) error {
	// 启动 DataHub
	if err := m.datahub.Start(); err != nil {
		return fmt.Errorf("failed to start datahub: %w", err)
	}

	// 启动 EventPersistService
	if err := m.eventPersist.Start(); err != nil {
		return fmt.Errorf("failed to start event persist: %w", err)
	}

	// 启动 AgentSDK
	if err := m.agentSDK.Start(); err != nil {
		return fmt.Errorf("failed to start agentsdk: %w", err)
	}

	// 启动 Gateway
	if err := m.gateway.Start(); err != nil {
		return fmt.Errorf("failed to start gateway: %w", err)
	}

	log.Println("[Master] Master 启动成功")
	log.Printf("[Master] Gateway 端口: %d, AgentSDK 端口: %d",
		config.GlobalConfig.Server.GatewayPort,
		config.GlobalConfig.Server.AgentSDKPort,
	)

	// 等待退出信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	// 优雅停止
	log.Println("[Master] 正在关闭...")
	return m.Stop()
}

// Stop 停止 Master
func (m *Master) Stop() error {
	// 停止 Gateway
	if err := m.gateway.Stop(); err != nil {
		log.Printf("[Master] 停止 Gateway 失败: %v", err)
	}

	// 停止 AgentSDK
	if err := m.agentSDK.Stop(); err != nil {
		log.Printf("[Master] 停止 AgentSDK 失败: %v", err)
	}

	// 停止 EventPersistService
	if err := m.eventPersist.Stop(); err != nil {
		log.Printf("[Master] 停止事件持久化服务失败: %v", err)
	}

	// 停止 DataHub
	if err := m.datahub.Stop(); err != nil {
		log.Printf("[Master] 停止 DataHub 失败: %v", err)
	}

	// 关闭数据库
	if err := m.database.Close(); err != nil {
		log.Printf("[Master] 关闭数据库失败: %v", err)
	}

	log.Println("[Master] Master 已停止")
	return nil
}

// DataHub 获取 DataHub 实例（供测试使用）
func (m *Master) DataHub() datahub.DataHub {
	return m.datahub
}

// Database 获取 Database 实例（供测试使用）
func (m *Master) Database() database.Database {
	return m.database
}

// Query 获取 Query 实例（供测试使用）
func (m *Master) Query() query.Query {
	return m.query
}

// Package atlhyper_master_v2 Master V2 核心包
//
// 本包提供 Master 的启动器，负责:
//   - 初始化所有依赖 (Store, CommandBus, Database, Processor, Query, CommandService, AgentSDK, Gateway)
//   - 依赖注入和组装
//   - 生命周期管理 (启动、运行、停止)
//
// 架构设计:
//   - 外部访问: Gateway → Query（读取）/ CommandService（写入）→ Store / CommandBus
//   - 内部处理: AgentSDK → Processor → Store; AgentSDK → CommandBus
//   - Gateway 禁止直接访问 Store
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
	"AtlHyper/atlhyper_master_v2/gateway"
	"AtlHyper/atlhyper_master_v2/mq"
	"AtlHyper/atlhyper_master_v2/processor"
	"AtlHyper/atlhyper_master_v2/service"
	"AtlHyper/atlhyper_master_v2/service/operations"
	"AtlHyper/atlhyper_master_v2/service/query"
)

// Master 是 Master V2 的主结构体
type Master struct {
	store        datahub.Store
	bus          mq.CommandBus
	database     database.Database
	processor    processor.Processor
	service      service.Service
	agentSDK     *agentsdk.Server
	gateway      *gateway.Server
	eventPersist *operations.EventPersistService
}

// New 创建并初始化 Master 实例
//
// 使用 config.GlobalConfig 中的配置初始化各层组件。
// 调用前必须先调用 config.LoadConfig()。
//
// 初始化顺序:
//  1. Store - 数据存储
//  2. CommandBus - 消息队列
//  3. Database - 持久化数据库
//  4. EventPersistService - Event 持久化服务
//  5. Processor - 数据处理层（写入 Store）
//  6. Query - 查询抽象层（读取 Store + CommandBus）
//  7. CommandService - 指令写入服务（写入 CommandBus）
//  8. AgentSDK - Agent 通信层
//  9. Gateway - Web API 网关
//
// 返回:
//   - *Master: Master 实例
//   - error: 初始化错误
func New() (*Master, error) {
	cfg := &config.GlobalConfig

	// 1. 初始化 Store (数据存储)
	store := datahub.New(datahub.Config{
		Type:            cfg.DataHub.Type,
		EventRetention:  cfg.DataHub.EventRetention,
		HeartbeatExpire: cfg.DataHub.HeartbeatExpire,
	})
	log.Printf("[Master] Store 初始化完成: type=%s", cfg.DataHub.Type)

	// 2. 初始化 CommandBus (消息队列)
	bus := mq.New(mq.Config{
		Type: cfg.DataHub.Type,
	})
	log.Println("[Master] CommandBus 初始化完成")

	// 3. 初始化 Database
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

	// 4. 初始化 EventPersistService
	eventPersist := operations.NewEventPersistService(
		store,
		db.ClusterEventRepository(),
		operations.EventPersistConfig{
			RetentionDays:   cfg.Event.RetentionDays,
			MaxCount:        cfg.Event.MaxCount,
			CleanupInterval: cfg.Event.CleanupInterval,
		},
	)
	log.Println("[Master] 事件持久化服务初始化完成")

	// 5. 初始化 Processor（写入路径）
	proc := processor.New(processor.Config{
		Store: store,
		OnSnapshotReceived: func(clusterID string) {
			// 触发 Event 持久化
			if err := eventPersist.Sync(clusterID); err != nil {
				log.Printf("[Master] 事件同步失败: 集群=%s, 错误=%v", clusterID, err)
			}
		},
	})
	log.Println("[Master] 数据处理器初始化完成")

	// 6. 初始化 Query（读取路径）
	q := query.NewWithEventRepo(store, bus, db.ClusterEventRepository())
	log.Println("[Master] 查询层初始化完成")

	// 7. 初始化 Operations（写入路径）
	ops := operations.NewCommandService(bus)
	log.Println("[Master] 操作服务初始化完成")

	// 组合统一 Service
	svc := service.New(q, ops)

	// 8. 初始化 AgentSDK（使用 Processor + Bus）
	agentServer := agentsdk.NewServer(agentsdk.Config{
		Port:           cfg.Server.AgentSDKPort,
		CommandTimeout: cfg.Timeout.CommandPoll,
		Bus:            bus,
		Processor:      proc,
	})
	log.Printf("[Master] AgentSDK 初始化完成: 端口=%d", cfg.Server.AgentSDKPort)

	// 9. 初始化 Gateway（使用统一 Service + Bus）
	gw := gateway.NewServer(gateway.Config{
		Port:     cfg.Server.GatewayPort,
		Service:  svc,
		Database: db,
		Bus:      bus,
	})
	log.Printf("[Master] Gateway 初始化完成: 端口=%d", cfg.Server.GatewayPort)

	return &Master{
		store:        store,
		bus:          bus,
		database:     db,
		processor:    proc,
		service:      svc,
		agentSDK:     agentServer,
		gateway:      gw,
		eventPersist: eventPersist,
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
	// 启动 Store
	if err := m.store.Start(); err != nil {
		return fmt.Errorf("failed to start store: %w", err)
	}

	// 启动 CommandBus
	if err := m.bus.Start(); err != nil {
		return fmt.Errorf("failed to start commandbus: %w", err)
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

	// 停止 CommandBus
	if err := m.bus.Stop(); err != nil {
		log.Printf("[Master] 停止 CommandBus 失败: %v", err)
	}

	// 停止 Store
	if err := m.store.Stop(); err != nil {
		log.Printf("[Master] 停止 Store 失败: %v", err)
	}

	// 关闭数据库
	if err := m.database.Close(); err != nil {
		log.Printf("[Master] 关闭数据库失败: %v", err)
	}

	log.Println("[Master] Master 已停止")
	return nil
}

// Store 获取 Store 实例（供测试使用）
func (m *Master) Store() datahub.Store {
	return m.store
}

// Bus 获取 CommandBus 实例（供测试使用）
func (m *Master) Bus() mq.CommandBus {
	return m.bus
}

// Database 获取 Database 实例（供测试使用）
func (m *Master) Database() database.Database {
	return m.database
}

// Service 获取 Service 实例（供测试使用）
func (m *Master) Service() service.Service {
	return m.service
}

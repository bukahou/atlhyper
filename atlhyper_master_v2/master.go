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
	"os"
	"os/signal"
	"syscall"

	"AtlHyper/atlhyper_master_v2/agentsdk"
	"AtlHyper/atlhyper_master_v2/ai"
	"AtlHyper/atlhyper_master_v2/config"
	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/database/repo"
	"AtlHyper/atlhyper_master_v2/database/sqlite"
	"AtlHyper/atlhyper_master_v2/datahub"
	"AtlHyper/atlhyper_master_v2/gateway"
	"AtlHyper/atlhyper_master_v2/mq"
	"AtlHyper/atlhyper_master_v2/notifier"
	"AtlHyper/atlhyper_master_v2/notifier/trigger"
	"AtlHyper/atlhyper_master_v2/processor"
	"AtlHyper/atlhyper_master_v2/service"
	"AtlHyper/atlhyper_master_v2/service/operations"
	"AtlHyper/atlhyper_master_v2/service/query"
	"AtlHyper/atlhyper_master_v2/service/sync"
	"AtlHyper/atlhyper_master_v2/slo"
	"AtlHyper/atlhyper_master_v2/tester"
	"AtlHyper/common/logger"
)

var log = logger.Module("Master")

// Master 是 Master V2 的主结构体
type Master struct {
	store        datahub.Store
	bus          mq.CommandBus
	database     *database.DB
	processor    processor.Processor
	service      service.Service
	agentSDK     *agentsdk.Server
	gateway      *gateway.Server
	testerServer *tester.Server
	eventPersist   *sync.EventPersistService
	metricsPersist *sync.MetricsPersistService
	alertManager   notifier.AlertManager
	heartbeat    *trigger.HeartbeatTrigger
	eventTrigger *trigger.EventTrigger
	// SLO 组件（始终启用）
	sloProcessor  *slo.Processor
	sloAggregator *slo.Aggregator
	sloCleaner    *slo.Cleaner
}

// New 创建并初始化 Master 实例
func New() (*Master, error) {
	cfg := &config.GlobalConfig

	// 1. 初始化 Store (数据存储)
	store := datahub.New(datahub.Config{
		Type:            cfg.DataHub.Type,
		EventRetention:  cfg.DataHub.EventRetention,
		HeartbeatExpire: cfg.DataHub.HeartbeatExpire,
		RedisAddr:       cfg.Redis.Addr,
		RedisPassword:   cfg.Redis.Password,
		RedisDB:         cfg.Redis.DB,
	})
	log.Info("Store 初始化完成", "type", cfg.DataHub.Type)

	// 2. 初始化 CommandBus (消息队列)
	bus := mq.New(mq.Config{
		Type:          cfg.DataHub.Type,
		RedisAddr:     cfg.Redis.Addr,
		RedisPassword: cfg.Redis.Password,
		RedisDB:       cfg.Redis.DB,
	})
	log.Info("CommandBus 初始化完成")

	// 3. 初始化 Database
	dialect := sqlite.NewDialect()
	db, err := database.New(database.Config{
		Type: cfg.Database.Type,
		Path: cfg.Database.Path,
	}, dialect)
	if err != nil {
		return nil, fmt.Errorf("failed to init database: %w", err)
	}

	// 设置 API Key 加密密钥（使用 JWT Secret）
	if err := repo.SetEncryptionSecret(cfg.JWT.SecretKey); err != nil {
		return nil, fmt.Errorf("failed to set encryption secret: %w", err)
	}

	repo.Init(db, dialect)
	log.Info("数据库初始化完成", "type", cfg.Database.Type)

	// 3.1 迁移旧 AI 配置到新表（如有）
	if err := database.MigrateOldAIConfig(context.Background(), db); err != nil {
		log.Warn("AI 配置迁移失败", "err", err)
	}

	// 3.2 初始化 AI Active Config（首次启动，从配置文件读取默认值）
	if err := database.InitAIActiveConfig(context.Background(), db, &cfg.AI); err != nil {
		log.Warn("AI Active Config 初始化失败", "err", err)
	}

	// 4. 初始化 EventPersistService
	eventPersist := sync.NewEventPersistService(
		store,
		db.Event,
		sync.EventPersistConfig{
			RetentionDays:   cfg.Event.RetentionDays,
			MaxCount:        cfg.Event.MaxCount,
			CleanupInterval: cfg.Event.CleanupInterval,
		},
	)
	log.Info("事件持久化服务初始化完成")

	// 4.1 初始化 MetricsPersistService
	metricsPersist := sync.NewMetricsPersistService(
		store,
		db.NodeMetrics,
		sync.MetricsPersistConfig{
			SampleInterval:  config.GlobalConfig.MetricsPersist.SampleInterval,
			RetentionDays:   config.GlobalConfig.MetricsPersist.RetentionDays,
			CleanupInterval: config.GlobalConfig.MetricsPersist.CleanupInterval,
		},
	)
	log.Info("节点指标持久化服务初始化完成")

	// 4.2 初始化 SLO 组件（始终启用，无需配置开关）
	sloProcessor := slo.NewProcessor(db.SLO)
	sloAggregator := slo.NewAggregator(db.SLO, cfg.SLO.AggregateInterval)
	sloCleaner := slo.NewCleaner(db.SLO, slo.CleanerConfig{
		RawRetention:    cfg.SLO.RawRetention,
		HourlyRetention: cfg.SLO.HourlyRetention,
		StatusRetention: cfg.SLO.StatusRetention,
		Interval:        cfg.SLO.CleanupInterval,
	})
	log.Info("SLO 组件初始化完成")

	// 5. 初始化 Processor（写入路径）
	proc := processor.New(processor.Config{
		Store: store,
		OnSnapshotReceived: func(clusterID string) {
			// 同步事件到数据库
			if err := eventPersist.Sync(clusterID); err != nil {
				log.Error("事件同步失败", "cluster", clusterID, "err", err)
			}
			// 同步节点指标到数据库
			if err := metricsPersist.Sync(clusterID); err != nil {
				log.Error("节点指标同步失败", "cluster", clusterID, "err", err)
			}
		},
	})
	log.Info("数据处理器初始化完成")

	// 6. 初始化 Query（读取路径）
	q := query.NewWithEventRepo(store, bus, db.Event)
	log.Info("查询层初始化完成")

	// 7. 初始化 Operations（写入路径）
	ops := operations.NewCommandService(bus, db.Command)
	log.Info("操作服务初始化完成")

	// 组合统一 Service
	svc := service.New(q, ops)

	// 8. 初始化 AgentSDK
	agentServer := agentsdk.NewServer(agentsdk.Config{
		Port:           cfg.Server.AgentSDKPort,
		CommandTimeout: cfg.Timeout.CommandPoll,
		Bus:            bus,
		Processor:      proc,
		SLOProcessor:   sloProcessor,
		CmdRepo:        db.Command,
	})
	log.Info("AgentSDK 初始化完成", "port", cfg.Server.AgentSDKPort)

	// 9. 初始化 AIService
	// AI 配置从 ai_providers + ai_active_config 表动态获取，支持热更新
	// 不再在启动时检查配置，Chat 时会实时从 DB 读取最新配置
	aiService := ai.NewService(
		ai.ServiceConfig{
			ToolTimeout: cfg.AI.ToolTimeout,
		},
		ops, bus,
		db.AIProvider, db.AIActive,
		db.AIConversation, db.AIMessage,
	)
	log.Info("AI 服务初始化完成 (动态配置)")

	// 10. 初始化 AlertManager（告警管理器）
	alertMgr, err := notifier.NewManager(db.Notify)
	if err != nil {
		return nil, fmt.Errorf("failed to init alert manager: %w", err)
	}
	log.Info("告警管理器初始化完成")

	// 11. 初始化 HeartbeatTrigger（心跳检测触发器）
	heartbeat := trigger.NewHeartbeatTrigger(store, alertMgr, trigger.HeartbeatConfig{
		CheckInterval: cfg.DataHub.HeartbeatExpire / 2,
		OfflineAfter:  cfg.DataHub.HeartbeatExpire,
	})
	log.Info("心跳检测触发器初始化完成")

	// 11.1 初始化 EventTrigger（事件告警触发器，可选）
	var eventTrigger *trigger.EventTrigger
	if cfg.EventAlert.Enabled {
		eventTrigger = trigger.NewEventTrigger(
			db.Event,
			q,
			alertMgr,
			trigger.EventConfig{
				CheckInterval: cfg.EventAlert.CheckInterval,
			},
		)
		log.Info("事件告警触发器初始化完成")
	}

	// 12. 初始化 Gateway
	gw := gateway.NewServer(gateway.Config{
		Port:      cfg.Server.GatewayPort,
		Service:   svc,
		Database:  db,
		Bus:       bus,
		AIService: aiService,
	})
	log.Info("Gateway 初始化完成", "port", cfg.Server.GatewayPort)

	// 13. 初始化 Tester
	testerServer := tester.NewServer(tester.Config{
		Port:         cfg.Server.TesterPort,
		AlertManager: alertMgr,
	})
	log.Info("Tester 初始化完成", "port", cfg.Server.TesterPort)

	return &Master{
		store:          store,
		bus:            bus,
		database:       db,
		processor:      proc,
		service:        svc,
		agentSDK:       agentServer,
		gateway:        gw,
		testerServer:   testerServer,
		eventPersist:   eventPersist,
		metricsPersist: metricsPersist,
		alertManager:   alertMgr,
		heartbeat:      heartbeat,
		eventTrigger:   eventTrigger,
		sloProcessor:   sloProcessor,
		sloAggregator:  sloAggregator,
		sloCleaner:     sloCleaner,
	}, nil
}

// Run 运行 Master
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

	// 启动 MetricsPersistService
	if err := m.metricsPersist.Start(); err != nil {
		return fmt.Errorf("failed to start metrics persist: %w", err)
	}

	// 启动 AlertManager
	if err := m.alertManager.Start(); err != nil {
		return fmt.Errorf("failed to start alert manager: %w", err)
	}

	// 启动 HeartbeatTrigger
	if err := m.heartbeat.Start(); err != nil {
		return fmt.Errorf("failed to start heartbeat trigger: %w", err)
	}

	// 启动 EventTrigger
	if m.eventTrigger != nil {
		if err := m.eventTrigger.Start(); err != nil {
			return fmt.Errorf("failed to start event trigger: %w", err)
		}
	}

	// 启动 SLO Aggregator
	m.sloAggregator.Start()
	log.Info("SLO 聚合器已启动")

	// 启动 SLO Cleaner
	m.sloCleaner.Start()
	log.Info("SLO 清理器已启动")

	// 启动 AgentSDK
	if err := m.agentSDK.Start(); err != nil {
		return fmt.Errorf("failed to start agentsdk: %w", err)
	}

	// 启动 Gateway
	if err := m.gateway.Start(); err != nil {
		return fmt.Errorf("failed to start gateway: %w", err)
	}

	// 启动 Tester
	if err := m.testerServer.Start(); err != nil {
		return fmt.Errorf("failed to start tester: %w", err)
	}

	log.Info("Master 启动成功",
		"gateway_port", config.GlobalConfig.Server.GatewayPort,
		"agentsdk_port", config.GlobalConfig.Server.AgentSDKPort,
	)

	// 等待退出信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	// 优雅停止
	log.Info("正在关闭...")
	return m.Stop()
}

// Stop 停止 Master
func (m *Master) Stop() error {
	// 停止 Tester
	if err := m.testerServer.Stop(); err != nil {
		log.Error("停止 Tester 失败", "err", err)
	}

	// 停止 Gateway
	if err := m.gateway.Stop(); err != nil {
		log.Error("停止 Gateway 失败", "err", err)
	}

	// 停止 AgentSDK
	if err := m.agentSDK.Stop(); err != nil {
		log.Error("停止 AgentSDK 失败", "err", err)
	}

	// 停止 HeartbeatTrigger
	if err := m.heartbeat.Stop(); err != nil {
		log.Error("停止心跳检测触发器失败", "err", err)
	}

	// 停止 EventTrigger
	if m.eventTrigger != nil {
		if err := m.eventTrigger.Stop(); err != nil {
			log.Error("停止事件告警触发器失败", "err", err)
		}
	}

	// 停止 SLO Cleaner
	m.sloCleaner.Stop()
	log.Info("SLO 清理器已停止")

	// 停止 SLO Aggregator
	m.sloAggregator.Stop()
	log.Info("SLO 聚合器已停止")

	// 停止 AlertManager
	m.alertManager.Stop()

	// 停止 EventPersistService
	if err := m.eventPersist.Stop(); err != nil {
		log.Error("停止事件持久化服务失败", "err", err)
	}

	// 停止 MetricsPersistService
	if err := m.metricsPersist.Stop(); err != nil {
		log.Error("停止节点指标持久化服务失败", "err", err)
	}

	// 停止 CommandBus
	if err := m.bus.Stop(); err != nil {
		log.Error("停止 CommandBus 失败", "err", err)
	}

	// 停止 Store
	if err := m.store.Stop(); err != nil {
		log.Error("停止 Store 失败", "err", err)
	}

	// 关闭数据库
	if err := m.database.Close(); err != nil {
		log.Error("关闭数据库失败", "err", err)
	}

	log.Info("Master 已停止")
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
func (m *Master) Database() *database.DB {
	return m.database
}

// Service 获取 Service 实例（供测试使用）
func (m *Master) Service() service.Service {
	return m.service
}

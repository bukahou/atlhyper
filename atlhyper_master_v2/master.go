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
//	master, err := atlhyper_master_v2.NewMaster()
//	master.Run(ctx)
package atlhyper_master_v2

import (
	aitools "AtlHyper/atlhyper_master_v2/ai/tools"
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"AtlHyper/atlhyper_master_v2/agentsdk"
	"AtlHyper/atlhyper_master_v2/ai"
	"AtlHyper/atlhyper_master_v2/aiops"
	aiopscore "AtlHyper/atlhyper_master_v2/aiops/core"
	"AtlHyper/atlhyper_master_v2/aiops/enricher"
	"AtlHyper/atlhyper_master_v2/auth/oidc"
	"AtlHyper/atlhyper_master_v2/config"
	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/database/repo"
	"AtlHyper/atlhyper_master_v2/database/sqlite"
	"AtlHyper/atlhyper_master_v2/datahub"
	"AtlHyper/atlhyper_master_v2/deployer"
	"AtlHyper/atlhyper_master_v2/gateway"
	"AtlHyper/atlhyper_master_v2/gateway/middleware"
	"AtlHyper/atlhyper_master_v2/github"
	"AtlHyper/atlhyper_master_v2/mq"
	"AtlHyper/atlhyper_master_v2/notifier"
	"AtlHyper/atlhyper_master_v2/notifier/pipeline"
	"AtlHyper/atlhyper_master_v2/processor"
	"AtlHyper/atlhyper_master_v2/service"
	"AtlHyper/atlhyper_master_v2/service/operations"
	"AtlHyper/atlhyper_master_v2/service/query"
	"AtlHyper/atlhyper_master_v2/service/sync"
	"AtlHyper/atlhyper_master_v2/slo"
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
	eventPersist *sync.EventPersistService
	alertManager notifier.AlertManager
	// AIOps 引擎
	aiopsEngine aiops.Engine
	// Deployer（GitOps CD）
	deployer deployer.Deployer
	// 只增不减的数据定期按保留期清理
	retentionJobs []retentionJob
}

// NewMaster 创建并初始化 Master 实例
func NewMaster() (*Master, error) {
	cfg := &config.GlobalConfig

	// 0. 校验 Agent 入口配置：认证模式写错时在打开任何资源之前就失败
	agentEntry, err := resolveAgentEntrySettings(cfg.AgentSDK)
	if err != nil {
		return nil, fmt.Errorf("Agent 入口配置无效: %w", err)
	}
	// 运维时区写错同样启动即失败（不静默退回 UTC）
	opsLocation, err := cfg.Server.Location()
	if err != nil {
		return nil, err
	}
	// 各集群最近一次上报是否带了有效令牌：Agent 入口写，集群管理页读
	agentAuthStatus := agentsdk.NewAuthStatusTracker()

	// 1. 初始化 Store (数据存储)
	store := datahub.NewStore(datahub.Config{
		Type:              cfg.DataHub.Type,
		EventRetention:    cfg.DataHub.EventRetention,
		HeartbeatExpire:   cfg.DataHub.HeartbeatExpire,
		SnapshotRetention: cfg.DataHub.SnapshotRetention,
		RedisAddr:         cfg.Redis.Addr,
		RedisPassword:     cfg.Redis.Password,
		RedisDB:           cfg.Redis.DB,
	})
	log.Info("Store 初始化完成", "type", cfg.DataHub.Type)

	// 2. 初始化 CommandBus (消息队列)
	bus := mq.NewCommandBus(mq.Config{
		Type:          cfg.DataHub.Type,
		RedisAddr:     cfg.Redis.Addr,
		RedisPassword: cfg.Redis.Password,
		RedisDB:       cfg.Redis.DB,
	})
	log.Info("CommandBus 初始化完成")

	// 3. 初始化 Database
	dialect := sqlite.NewDialect()
	db, err := database.NewDatabase(database.Config{
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

	// 3.1 初始化 AI Settings（首次启动，从配置文件读取默认值）
	if err := database.InitAISettings(context.Background(), db, &cfg.AI); err != nil {
		log.Warn("AI Settings 初始化失败", "err", err)
	}

	// 3.3 种子 AI Provider（从环境变量自动创建，仅首次无数据时生效）
	if err := database.SeedAIProvider(context.Background(), db, &cfg.AI.Seed); err != nil {
		log.Warn("AI Provider 种子初始化失败", "err", err)
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

	// 4.1 初始化 SLO 路由映射更新器（仅保留路由映射写入，时序数据已迁移至 OTelSnapshot + ClickHouse）
	sloRouteUpdater := slo.NewRouteUpdater(db.SLO)
	log.Info("SLO 路由映射更新器初始化完成")

	// 4.4 初始化 AIOps 引擎
	var aiopsEngine aiops.Engine
	aiopsEngine = aiopscore.NewEngine(aiopscore.EngineConfig{
		Store:         store,
		GraphRepo:     db.AIOpsGraph,
		BaselineRepo:  db.AIOpsBaseline,
		IncidentRepo:  db.AIOpsIncident,
		SLORepo:       db.SLO,
		FlushInterval: cfg.AIOps.FlushInterval,
	})
	log.Info("AIOps 引擎初始化完成")

	// 5. 初始化 Processor（写入路径）
	proc := processor.NewProcessor(processor.Config{
		Store: store,
		OnSnapshotReceived: func(clusterID string) {
			// 同步事件到数据库
			if err := eventPersist.Sync(clusterID); err != nil {
				log.Error("事件同步失败", "cluster", clusterID, "err", err)
			}
			// 更新 SLO 路由映射（时序数据已迁移，仅保留路由映射更新）
			if err := sloRouteUpdater.Sync(store, clusterID); err != nil {
				log.Error("SLO 路由映射更新失败", "cluster", clusterID, "err", err)
			}
			// AIOps 引擎处理
			aiopsEngine.OnSnapshot(clusterID)
		},
	})
	log.Info("数据处理器初始化完成")

	// 6. 初始化 Operations（写入路径，AI Service 依赖 cmdOps）
	cmdOps := operations.NewCommandService(bus, db.Command, storePresence{store: store})
	adminOps := operations.NewAdminService(db.AIProvider, db.AISettings, db.AIRoleBudget)
	log.Info("操作服务初始化完成")

	// 7. 初始化 AI Service（Enricher 依赖 AIService）
	aiService := ai.NewService(
		ai.ServiceConfig{BedrockRegion: cfg.AI.BedrockRegion},
		db.AIProvider, db.AIModel, db.AIRoleBudget,
		db.AIConversation, db.AIMessage,
	)
	log.Info("AI 服务初始化完成 (动态配置)")

	// 7.1 初始化 AIOps Enricher（通过 ai.AIService 接口调用 LLM，不再直接操作 ai/llm）
	aiopsEnricher := enricher.NewEnricher(db.AIOpsIncident, db.AIReport, aiService)
	aiopsEnricher.SetStore(store)          // OTel 上下文丰富：读取 OTelSnapshot
	aiopsEnricher.SetEventSource(db.Event) // 持久化 K8s event 进入分析上下文
	// 深度分析报告只有拥有 ai.analyze 的人能看：工具按这一级的权限授权，不多给
	analysisAudience := middleware.MinLevelFor(middleware.PermAIAnalyze)
	aiopsEnricher.SetAnalysisToolPolicy(func(p string) bool {
		return middleware.Allowed(analysisAudience, middleware.Permission(p))
	})
	aiopsEnricher.EnableBackgroundTrigger(db.AIRoleBudget, opsLocation)
	aiopsEngine.SubscribeTransitions(aiopsEnricher.OnIncidentTransition)
	log.Info("AIOps Enricher 初始化完成（后台自动分析已启用）")

	// 8. 初始化 Query（读取路径）— 全部依赖通过构造函数注入
	q := query.NewQueryService(query.QueryServiceDeps{
		Store:       store,
		Bus:         bus,
		EventRepo:   db.Event,
		SLORepo:     db.SLO,
		AIOpsEngine: aiopsEngine,
		AIOpsAI:     aiopsEnricher,
		AdminRepos: query.AdminRepos{
			Audit:      db.Audit,
			Command:    db.Command,
			AIProvider: db.AIProvider,
			AISettings: db.AISettings,
			AIModel:    db.AIModel,
			AIBudget:   db.AIRoleBudget,
			AIReport:   db.AIReport,
		},
		DeployRepos: query.DeployRepos{
			DeployConfig:  db.DeployConfig,
			DeployHistory: db.DeployHistory,
			GitHubInstall: db.GitHubInstall,
			RepoConfig:    db.RepoConfig,
		},
		UserRepo:     db.User,
		IdentityRepo: db.UserIdentity,

		ClusterRepo:       db.Cluster,
		ClusterAuthStatus: agentAuthStatus,

		NotifyRepos: query.NotifyRepos{
			Policy: db.NotifyPolicy, Alerts: db.NotifyAlert, Deliveries: db.NotifyDelivery,
			Receivers: db.NotifyReceiver, Rules: db.NotifyRule, Silences: db.NotifySilence,
			SMTP: db.NotifySMTP,
		},
	})
	log.Info("查询层初始化完成")

	// 初始化 SLO 写入服务
	sloOps := operations.NewSLOService(db.SLO)

	// 初始化部署 / GitHub 集成写入服务
	deployOps := operations.NewDeployService(db.DeployConfig, db.GitHubInstall)

	// 初始化用户写入服务
	userOps := operations.NewUserService(db.User, db.UserIdentity)

	// 初始化集群注册表写入服务（登记、签发 Agent 令牌）
	clusterRegistryOps := operations.NewClusterRegistryService(db.Cluster)

	// 初始化通知设置写入服务（策略、接收人、发件设置、路由规则、静默）
	notifyOps := operations.NewNotifyService(operations.NotifyRepos{
		Policy: db.NotifyPolicy, Receivers: db.NotifyReceiver, Rules: db.NotifyRule, Silences: db.NotifySilence,
		SMTP: db.NotifySMTP,
	})

	// 组合统一 Service
	svc := service.NewService(q, cmdOps, adminOps, sloOps, deployOps, userOps, clusterRegistryOps, notifyOps)

	// 8. 初始化 AgentSDK
	agentServer := agentsdk.NewServer(agentsdk.Config{
		Port:           cfg.Server.AgentSDKPort,
		CommandTimeout: cfg.Timeout.CommandPoll,
		Bus:            bus,
		Processor:      proc,
		CmdRepo:        db.Command,

		AuthMode:         agentEntry.authMode,
		Clusters:         db.Cluster,
		AuthStatus:       agentAuthStatus,
		MaxSnapshotBytes: agentEntry.maxSnapshotBytes,
	})
	log.Info("AgentSDK 初始化完成",
		"port", cfg.Server.AgentSDKPort,
		"authMode", agentEntry.authMode,
		"maxSnapshotMB", cfg.AgentSDK.MaxSnapshotMB)

	// 10. 初始化通知模块：告警来源（事件变化、Agent 心跳、K8s 事件）→ 管道 → 发送
	alertMgr, err := notifier.NewAlertManager(notifier.Config{
		Repos: pipeline.Repos{
			Policy: db.NotifyPolicy, Alerts: db.NotifyAlert, Deliveries: db.NotifyDelivery,
			Receivers: db.NotifyReceiver, Rules: db.NotifyRule, Silences: db.NotifySilence,
		},
		SMTP:                   db.NotifySMTP,
		Events:                 db.Event,
		Agents:                 store,
		PublicURL:              cfg.Server.PublicURL,
		Location:               opsLocation,
		HeartbeatCheckInterval: cfg.DataHub.HeartbeatExpire / 2,
		HeartbeatOfflineAfter:  cfg.DataHub.HeartbeatExpire,
		K8sEventCheckInterval:  cfg.EventAlert.CheckInterval,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to init alert manager: %w", err)
	}
	// 事件变化交给通知模块。回调只入队：引擎在持有状态机锁时调用它
	aiopsEngine.SubscribeTransitions(alertMgr.OnIncidentTransition)
	aiopsEngine.SubscribeLinks(alertMgr.OnIncidentLink)
	log.Info("通知模块初始化完成")

	// 11.5 初始化 GitHub Client（可选，未配置则跳过）
	var ghClient github.Client
	if cfg.GitHub.AppID > 0 && cfg.GitHub.PrivateKeyPath != "" {
		var err error
		ghClient, err = github.NewClient(github.Config{
			AppID:          cfg.GitHub.AppID,
			AppSlug:        cfg.GitHub.AppSlug,
			PrivateKeyPath: cfg.GitHub.PrivateKeyPath,
			CallbackURL:    cfg.GitHub.CallbackURL,
		})
		if err != nil {
			log.Warn("GitHub Client 初始化失败，GitHub 集成不可用", "err", err)
		} else {
			// 从数据库恢复 Installation ID
			inst, _ := db.GitHubInstall.Get(context.Background())
			if inst != nil {
				if setter, ok := ghClient.(interface{ SetInstallationID(int64) }); ok {
					setter.SetInstallationID(inst.InstallationID)
				}
				log.Info("GitHub Client 初始化完成", "installationID", inst.InstallationID)
			} else {
				log.Info("GitHub Client 初始化完成（未连接）")
			}
		}
	}

	// 11.6 注册 AI 工具（定义都在 ai/tools；可选依赖缺失的工具不注册，清单里也就没有它）
	toolDeps := aitools.Deps{
		Commands:    cmdOps,
		ToolTimeout: cfg.AI.ToolTimeout,
		AIOps:       aiopsEngine,
		SummarizeIncident: func(ctx context.Context, incidentID string) (any, error) {
			return aiopsEnricher.Summarize(ctx, incidentID)
		},
		Snapshots: store,
		Events:    db.Event,
	}
	if ghClient != nil {
		// 部署历史只在有部署器（依赖 GitHub）时才有；GitHub 工具只读公开仓库 + 显式列入白名单的私有仓库
		toolDeps.DeployHistory = db.DeployHistory
		toolDeps.GitHub = ghClient
		toolDeps.GitHubScope = aitools.NewGitHubRepoScope(ghClient, cfg.AI.GitHubPrivateRepos)
	}
	aitools.Register(aiService.Tools(), toolDeps)
	log.Info("AI 工具注册完成", "tools", aiService.Tools().Names())

	// 11.7 初始化 Deployer（GitOps CD，仅在 GitHub Client 可用时启用）
	var deployerService deployer.Deployer
	if ghClient != nil {
		deployerService = deployer.NewService(ghClient, db, bus)
		log.Info("Deployer 初始化完成")
	}

	// 12. 初始化 Gateway
	// 12.5 外部身份登录（OIDC，可选）：首登自动建为成员（0 级），由管理员提升
	var oidcModule *oidc.Module
	if cfg.OIDC.Enabled() {
		oidcModule = oidc.NewModule(oidc.Config{
			Issuer:         cfg.OIDC.Issuer,
			ClientID:       cfg.OIDC.ClientID,
			ClientSecret:   cfg.OIDC.ClientSecret,
			RedirectURL:    cfg.OIDC.RedirectURL,
			CookieSecret:   cfg.OIDC.CookieSecret,
			CookieSecure:   cfg.OIDC.CookieSecure,
			DisplayName:    cfg.OIDC.DisplayName,
			ClientIPHeader: cfg.Server.ClientIPHeader,
			NewUserRole:    middleware.RoleMember,
			Accounts:       svc,
			IssueToken:     middleware.GenerateToken,
		})
		log.Info("OIDC 登录已启用（首次使用时连接 issuer）", "issuer", cfg.OIDC.Issuer)
	}

	if cfg.Server.ClientIPHeader == "" {
		log.Warn("未配置 MASTER_GATEWAY_CLIENT_IP_HEADER：审计记录的地址与按地址限流只能看到入口代理，所有客户端共用一个限流额度")
	}
	gw := gateway.NewServer(gateway.Config{
		Port:           cfg.Server.GatewayPort,
		Service:        svc,
		Database:       db,
		AIService:      aiService,
		AnalyzeTrigger: aiopsEnricher,
		GitHubClient:   ghClient,
		Deployer:       deployerService,
		OIDC:           oidcModule,
		NotifyTester:   alertMgr, // 接收人测试发送走网关（同一套权限与审计）
	})
	log.Info("Gateway 初始化完成", "port", cfg.Server.GatewayPort)

	return &Master{
		store:        store,
		bus:          bus,
		database:     db,
		processor:    proc,
		service:      svc,
		agentSDK:     agentServer,
		gateway:      gw,
		eventPersist: eventPersist,
		alertManager: alertMgr,
		aiopsEngine:  aiopsEngine,
		deployer:     deployerService,

		retentionJobs: []retentionJob{
			{name: "指令历史", keep: cfg.Database.CommandHistoryRetention, purge: cmdOps.PurgeHistoryBefore},
			{name: "审计记录", keep: cfg.Database.AuditRetention, purge: db.Audit.DeleteBefore},
			{name: "已恢复的 AIOps 事件", keep: cfg.Database.AIOpsIncidentRetention, purge: db.AIOpsIncident.DeleteResolvedBefore},
			{name: "AI 分析报告", keep: cfg.Database.AIReportRetention, purge: db.AIReport.DeleteBefore},
		},
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

	// 启动通知模块（管道恢复状态 + 心跳与 K8s 事件两个轮询来源）；先于 AIOps 引擎，事件变化一发布就有人接
	if err := m.alertManager.Start(); err != nil {
		return fmt.Errorf("failed to start alert manager: %w", err)
	}

	// 启动 AIOps 引擎
	if m.aiopsEngine != nil {
		if err := m.aiopsEngine.Start(ctx); err != nil {
			return fmt.Errorf("failed to start aiops engine: %w", err)
		}
	}

	// 启动 Deployer
	if m.deployer != nil {
		if err := m.deployer.Start(ctx); err != nil {
			log.Warn("Deployer 启动失败", "err", err)
		}
	}

	// 只增不减的数据按保留期定期清理（随 ctx 结束）
	go runRetention(ctx, m.retentionJobs)

	// 启动 AgentSDK
	if err := m.agentSDK.Start(); err != nil {
		return fmt.Errorf("failed to start agentsdk: %w", err)
	}

	// 启动 Gateway
	if err := m.gateway.Start(); err != nil {
		return fmt.Errorf("failed to start gateway: %w", err)
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
	// 停止 Gateway
	if err := m.gateway.Stop(); err != nil {
		log.Error("停止 Gateway 失败", "err", err)
	}

	// 停止 AgentSDK
	if err := m.agentSDK.Stop(); err != nil {
		log.Error("停止 AgentSDK 失败", "err", err)
	}

	// 停止 AIOps 引擎
	if m.aiopsEngine != nil {
		if err := m.aiopsEngine.Stop(); err != nil {
			log.Error("停止 AIOps 引擎失败", "err", err)
		}
	}

	// 停止 Deployer
	if m.deployer != nil {
		if err := m.deployer.Stop(); err != nil {
			log.Error("停止 Deployer 失败", "err", err)
		}
	}

	// 停止通知模块（两个轮询来源与管道）
	m.alertManager.Stop()

	// 停止 EventPersistService
	if err := m.eventPersist.Stop(); err != nil {
		log.Error("停止事件持久化服务失败", "err", err)
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

// cmd/atlhyper_master/main.go
package main

import (
	"context"
	"log"
	"strings"

	"AtlHyper/atlhyper_master/config"
	"AtlHyper/atlhyper_master/gateway"
	"AtlHyper/atlhyper_master/repository"
	"AtlHyper/atlhyper_master/repository/mem"
	reposql "AtlHyper/atlhyper_master/repository/sql"
	"AtlHyper/atlhyper_master/store"
	"AtlHyper/atlhyper_master/store/memory"
	"AtlHyper/atlhyper_master/store/sqlite"

	external "AtlHyper/atlhyper_master"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func main() {
	// ========================================
	// Phase 1: 基础设施初始化（必须按顺序）
	// ========================================

	// 1. 加载配置（最先执行，其他模块依赖配置）
	config.LoadConfig()

	// 2. 设置结构化日志
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	// 3. 初始化存储层
	// 3.1 SQLite 存储引擎
	db, err := sqlite.Init("")
	if err != nil {
		log.Fatalf("❌ SQLite 初始化失败: %v", err)
	}
	store.SetDB(db)

	// 3.2 内存存储引擎
	memory.Bootstrap()

	// 4. 初始化仓库层
	// 4.1 SQL 仓库
	reposql.Init()

	// 4.2 内存仓库
	mem.Init()

	// 5. 验证仓库注册
	if err := repository.Validate(); err != nil {
		log.Fatalf("❌ 仓库注册验证失败: %v", err)
	}

	// 6. 初始化默认数据
	ctx := context.Background()
	if err := reposql.EnsureAdminUser(ctx); err != nil {
		log.Printf("⚠️ 初始化管理员失败: %v", err)
	}
	if err := reposql.EnsureDefaultUsers(ctx); err != nil {
		log.Printf("⚠️ 初始化默认用户失败: %v", err)
	}
	if err := initNotifyConfig(ctx); err != nil {
		log.Printf("⚠️ 初始化通知配置失败: %v", err)
	}

	log.Println("✅ 核心基础设施初始化完成")

	// ========================================
	// Phase 2: 可选功能模块初始化
	// ========================================
	external.StartOptionalServices()

	// ========================================
	// Phase 3: 启动 HTTP Server（阻塞主线程）
	// ========================================
	log.Println("🌐 启动 HTTP Server（UI API + Ingest + Control）")
	gateway.StartHTTPServer()
}

// initNotifyConfig 初始化通知配置表
func initNotifyConfig(ctx context.Context) error {
	cfg := config.GlobalConfig

	// Slack 配置
	slackInterval := int64(cfg.Slack.DispatchInterval.Seconds())
	if slackInterval <= 0 {
		slackInterval = 5
	}

	// Mail 配置
	mailInterval := int64(cfg.Diagnosis.AlertDispatchInterval.Seconds())
	if mailInterval <= 0 {
		mailInterval = 60
	}
	mailTo := strings.Join(cfg.Mailer.To, ",")

	return reposql.InitNotifyTables(ctx,
		cfg.Slack.WebhookURL, cfg.Slack.EnableSlackAlert, slackInterval,
		cfg.Mailer.SMTPHost, cfg.Mailer.SMTPPort, cfg.Mailer.Username,
		cfg.Mailer.Password, cfg.Mailer.From, mailTo,
		cfg.Mailer.EnableEmailAlert, mailInterval)
}

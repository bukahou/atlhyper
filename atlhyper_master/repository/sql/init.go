// repository/sql/init.go
//
// SQL 仓库初始化
//
// 本文件负责 SQL 仓库层的初始化工作，包括:
//   1. 根据数据库类型加载对应的 SQL 语句集
//   2. 注册各业务仓库实现到全局注册表
//   3. 初始化默认管理员账户和用户
//   4. 初始化通知配置表的默认记录
//
// 调用顺序:
//   main.go -> bootstrap.go -> sql.Init() -> sql.EnsureAdminUser() -> sql.EnsureDefaultUsers()
//
// 依赖关系:
//   - store.DB: 数据库连接 (需要在调用前初始化)
//   - repository: 仓库接口注册表
//   - config: 全局配置 (管理员账户信息)
package sql

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"AtlHyper/atlhyper_master/config"
	"AtlHyper/atlhyper_master/model/entity"
	"AtlHyper/atlhyper_master/repository"
	"AtlHyper/atlhyper_master/repository/sql/sqlite"
	"AtlHyper/atlhyper_master/store"

	"golang.org/x/crypto/bcrypt"
)

// ============================================================================
// Init SQL 仓库初始化入口
// ============================================================================

// Init 初始化 SQL 仓库并注册到全局
//
// 该函数是 SQL 仓库层的入口点，执行以下操作:
//   1. 根据 dbType 参数加载对应数据库的 SQL 语句
//   2. 创建并注册各业务仓库实现
//
// 参数:
//   - dbType: 可变参数，指定数据库类型
//   - 支持值: "sqlite" (默认), "postgres"
//   - 如果未提供或为空，默认使用 "sqlite"
//
// 使用示例:
//
//	sql.Init()           // 使用默认 SQLite
//	sql.Init("sqlite")   // 显式指定 SQLite
//	sql.Init("postgres") // 使用 PostgreSQL (需要 pg 驱动)
//
// 注意:
//   - 必须在 store.DB 初始化之后调用
//   - 必须在任何数据库操作之前调用
func Init(dbType ...string) {
	// 1. 确定数据库类型，默认为 sqlite
	dt := "sqlite"
	if len(dbType) > 0 && dbType[0] != "" {
		dt = dbType[0]
	}

	// 2. 根据数据库类型加载对应的 SQL 语句集
	switch dt {
	case "sqlite":
		// SQLite: 使用 ? 占位符，LastInsertId() 获取ID
		SetQueries(sqlite.Queries)
	case "postgres":
		// PostgreSQL: 使用 $1, $2 占位符，RETURNING id 获取ID
		// TODO: 完成 PostgreSQL 支持后取消注释
		// SetQueries(pg.Queries)
		SetQueries(sqlite.Queries) // 暂时回退到 sqlite
		log.Println("⚠️ PostgreSQL 尚未完全实现，使用 SQLite 查询")
	default:
		// 未知类型，回退到 SQLite
		SetQueries(sqlite.Queries)
	}

	// 3. 注册仓库实现到全局注册表
	// 这使得其他模块可以通过 repository.User、repository.Audit 等访问
	repository.InitSQL(
		&UserRepo{},    // 用户仓库
		&AuditRepo{},   // 审计日志仓库
		&EventRepo{},   // 事件日志仓库
		&ConfigRepo{},  // 配置仓库
		&MetricsRepo{}, // 指标仓库
	)
	log.Printf("✅ SQL 仓库初始化完成 (数据库类型: %s)", dt)
}

// ============================================================================
// EnsureAdminUser 管理员账户初始化
// ============================================================================

// EnsureAdminUser 初始化默认管理员账户
//
// 该函数检查用户表是否为空，如果为空则创建默认管理员账户。
// 管理员信息从 config.GlobalConfig.Admin 读取。
//
// 执行逻辑:
//   1. 统计用户表记录数
//   2. 如果已有用户，跳过初始化
//   3. 如果用户表为空，使用配置文件中的信息创建管理员
//
// 参数:
//   - ctx: 上下文，用于数据库操作超时控制
//
// 返回:
//   - error: 创建失败时返回错误
//
// 配置项 (config.yaml):
//
//	admin:
//	  username: admin
//	  password: admin123
//	  display_name: Administrator
//	  email: admin@example.com
//	  role: 1
func EnsureAdminUser(ctx context.Context) error {
	// 检查是否已有用户
	count, err := repository.User.Count(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		log.Println("ℹ️ 用户表已存在用户，跳过管理员初始化")
		return nil
	}

	// 从配置文件读取管理员信息
	cfg := config.GlobalConfig.Admin
	username := cfg.Username
	password := cfg.Password
	displayName := cfg.DisplayName
	email := cfg.Email

	// 对密码进行 bcrypt 哈希
	// bcrypt.DefaultCost = 10，平衡安全性和性能
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	// 解析角色，默认为 1 (管理员)
	role := 1
	if parsed, err := strconv.Atoi(cfg.Role); err == nil {
		role = parsed
	}

	// 创建管理员用户
	_, err = repository.User.Insert(ctx, &entity.User{
		Username:     username,
		PasswordHash: string(hashed),
		DisplayName:  displayName,
		Email:        email,
		Role:         role,
		CreatedAt:    time.Now().UTC(),
	})
	if err != nil {
		return err
	}

	log.Printf("✅ 默认管理员已创建: 用户名 %s / 角色 %d", username, role)
	return nil
}

// ============================================================================
// EnsureDefaultUsers 默认用户批量初始化
// ============================================================================

// EnsureDefaultUsers 初始化环境变量中配置的默认用户
//
// 该函数从环境变量读取用户配置，批量创建默认用户。
// 主要用于开发/测试环境快速初始化多个测试账户。
//
// 环境变量格式:
//
//	DEFAULT_USER_COUNT=2
//	USER_1_USERNAME=user1
//	USER_1_PASSWORD=pass1
//	USER_1_DISPLAY_NAME=User One
//	USER_1_EMAIL=user1@example.com
//	USER_1_ROLE=2
//	USER_2_USERNAME=user2
//	...
//
// 执行逻辑:
//   1. 读取 DEFAULT_USER_COUNT 确定用户数量
//   2. 遍历每个用户配置
//   3. 跳过已存在的用户名
//   4. 创建新用户
//
// 参数:
//   - ctx: 上下文
//
// 返回:
//   - error: 目前总是返回 nil，错误仅记录日志
func EnsureDefaultUsers(ctx context.Context) error {
	// 读取用户数量
	countStr := os.Getenv("DEFAULT_USER_COUNT")
	if countStr == "" {
		return nil
	}
	n, err := strconv.Atoi(countStr)
	if err != nil || n <= 0 {
		return nil
	}

	// 遍历每个用户配置
	for i := 1; i <= n; i++ {
		prefix := "USER_" + strconv.Itoa(i) + "_"

		// 读取用户信息
		username := os.Getenv(prefix + "USERNAME")
		password := os.Getenv(prefix + "PASSWORD")
		displayName := os.Getenv(prefix + "DISPLAY_NAME")
		email := os.Getenv(prefix + "EMAIL")
		roleStr := os.Getenv(prefix + "ROLE")

		// 验证必填字段
		if username == "" || password == "" || email == "" {
			log.Printf("⚠️ 用户 %d 信息不完整，跳过", i)
			continue
		}

		// 检查用户名是否已存在
		exists, _ := repository.User.ExistsByUsername(ctx, username)
		if exists {
			log.Printf("ℹ️ 用户 %q 已存在，跳过创建", username)
			continue
		}

		// 解析角色
		role := 1
		if parsed, err := strconv.Atoi(roleStr); err == nil {
			role = parsed
		}

		// 密码哈希
		hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			log.Printf("❌ 加密密码失败（用户 %q）: %v", username, err)
			continue
		}

		// 创建用户
		_, err = repository.User.Insert(ctx, &entity.User{
			Username:     username,
			PasswordHash: string(hashed),
			DisplayName:  displayName,
			Email:        email,
			Role:         role,
			CreatedAt:    time.Now().UTC(),
		})
		if err != nil {
			log.Printf("❌ 创建用户 %q 失败: %v", username, err)
			continue
		}

		log.Printf("✅ 创建用户 %d: 用户名=%q，角色=%d", i, username, role)
	}
	return nil
}

// ============================================================================
// InitNotifyTables 通知配置表初始化
// ============================================================================

// InitNotifyTables 初始化通知配置表的默认记录
//
// 该函数检查 Slack 和邮件通知配置表，如果记录不存在则创建默认配置。
// 配置表采用单行模式 (id=1)，每种通知类型只有一条配置记录。
//
// 参数:
//   - ctx: 上下文
//   - slackWebhook: Slack Webhook URL
//   - slackEnable: 是否启用 Slack 通知
//   - slackInterval: Slack 通知发送间隔 (秒)
//   - mailHost: SMTP 服务器地址
//   - mailPort: SMTP 端口
//   - mailUser: SMTP 用户名
//   - mailPass: SMTP 密码
//   - mailFrom: 发件人地址
//   - mailTo: 收件人地址 (多个用逗号分隔)
//   - mailEnable: 是否启用邮件通知
//   - mailInterval: 邮件通知发送间隔 (秒)
//
// 返回:
//   - error: 初始化失败时返回错误
//
// 注意:
//   - 该函数只在记录不存在时插入，不会更新已有记录
//   - 直接使用 store.DB 而非 repository，因为 repository 是更高层抽象
func InitNotifyTables(ctx context.Context, slackWebhook string, slackEnable bool, slackInterval int64,
	mailHost, mailPort, mailUser, mailPass, mailFrom, mailTo string, mailEnable bool, mailInterval int64) error {

	db := store.DB

	// -------------------- 初始化 Slack 配置 --------------------
	var slackExists int
	db.QueryRowContext(ctx, Q.Config.CountSlack).Scan(&slackExists)
	if slackExists == 0 {
		// 将布尔值转换为整数 (SQLite 兼容)
		enableInt := 0
		if slackEnable {
			enableInt = 1
		}
		_, err := db.ExecContext(ctx, Q.Config.InsertSlack,
			enableInt,
			slackWebhook,
			slackInterval,
			time.Now().Format(time.RFC3339))
		if err != nil {
			return fmt.Errorf("init slack config: %w", err)
		}
	}

	// -------------------- 初始化邮件配置 --------------------
	var mailExists int
	db.QueryRowContext(ctx, Q.Config.CountMail).Scan(&mailExists)
	if mailExists == 0 {
		enableInt := 0
		if mailEnable {
			enableInt = 1
		}
		_, err := db.ExecContext(ctx, Q.Config.InsertMail,
			enableInt,
			mailHost,
			mailPort,
			mailUser,
			mailPass,
			mailFrom,
			mailTo,
			mailInterval,
			time.Now().Format(time.RFC3339))
		if err != nil {
			return fmt.Errorf("init mail config: %w", err)
		}
	}

	return nil
}

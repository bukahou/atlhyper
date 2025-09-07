package config

import (
	"AtlHyper/atlhyper_master/db/utils"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// 环境变量解析 —— 读不到就用默认值
func envBoolDefault(key string, def int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" { return def }
	switch strings.ToLower(v) {
	case "1", "true", "t", "yes", "y", "on":
		return 1
	default:
		return 0
	}
}
func envInt64Default(key string, def int64) int64 {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" { return def }
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil || n <= 0 { return def }
	return n
}
func envStringDefault(key, def string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" { return def }
	return v
}

// 只在不存在记录时，用环境变量写入一行初始配置
func InitConfigTables() error {
	var exists int
	if err := utils.DB.QueryRow(
		`SELECT COUNT(*) FROM config WHERE id = 1`,
	).Scan(&exists); err != nil {
		return err
	}
	if exists > 0 {
		fmt.Println("ℹ️ config 表已有初始化记录，跳过")
		return nil
	}

	enable := envBoolDefault("ENABLE_SLACK_ALERT", 0)                // 读不到 ⇒ 0
	webhook := envStringDefault("SLACK_WEBHOOK_URL", "")             // 读不到 ⇒ ""
	intervalSec := envInt64Default("SLACK_DISPATCH_INTERVAL_SEC", 5) // 读不到 ⇒ 5

	_, err := utils.DB.Exec(`
		INSERT INTO config (id, name, enable, webhook, interval_sec, updated_at)
		VALUES (1, ?, ?, ?, ?, ?)`,
		"slack", enable, webhook, intervalSec, time.Now().Format(time.RFC3339),
	)
	if err != nil {
		return err
	}
	fmt.Println("✅ slack 配置初始化完成（来源：环境变量）")
	return nil
}
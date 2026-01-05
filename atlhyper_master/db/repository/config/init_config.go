package config

import (
	"fmt"
	"time"

	"AtlHyper/atlhyper_master/config"
	"AtlHyper/atlhyper_master/db/utils"
)

// InitConfigTables 只在不存在记录时，用全局配置写入一行初始配置
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

	// 从全局配置读取 Slack 设置
	enable := 0
	if config.GlobalConfig.Slack.EnableSlackAlert {
		enable = 1
	}
	webhook := config.GlobalConfig.Slack.WebhookURL
	intervalSec := int64(config.GlobalConfig.Slack.DispatchInterval / time.Second)

	_, err := utils.DB.Exec(`
		INSERT INTO config (id, name, enable, webhook, interval_sec, updated_at)
		VALUES (1, ?, ?, ?, ?, ?)`,
		"slack", enable, webhook, intervalSec, time.Now().Format(time.RFC3339),
	)
	if err != nil {
		return err
	}
	fmt.Println("✅ slack 配置初始化完成（来源：全局配置）")
	return nil
}
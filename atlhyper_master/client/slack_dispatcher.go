// package client

// import (
// 	"AtlHyper/atlhyper_master/client/slack"
// 	"AtlHyper/atlhyper_master/config"
// 	"log"
// 	"time"
// )
// func StartSlackDispatcher() {
// 	// 🚫 若配置中未启用 Slack 告警功能，则直接退出
// 	if !config.GlobalConfig.Slack.EnableSlackAlert {
// 		log.Println("⚠️ Slack 告警功能已关闭，未启动调度器。")
// 		return
// 	}

// 	// 🕒 获取配置中的告警发送间隔
// 	interval := config.GlobalConfig.Slack.DispatchInterval

// 	// ✅ 启动后台 goroutine，周期性处理告警调度任务
// 	go func() {
// 		for {
// 			// 🚀 调用 Slack 模块进行告警发送（从清洗后的事件池中）
// 			slack.DispatchSlackAlertFromCleanedEvents()

// 			// ⏱ 间隔等待下一轮告警处理
// 			time.Sleep(interval)
// 		}
// 	}()

// 	log.Println("✅ Slack 告警调度器启动成功。")
// }

// atlhyper_master/client/dispatcher/slack_dispatcher.go (或保留原文件名)
package client

import (
	"AtlHyper/atlhyper_master/client/slack"
	"AtlHyper/atlhyper_master/service/config"
	"log"
	"strings"
	"sync"
	"time"
)

const (
	disabledPollInterval = 10 * time.Second // 关闭/未配置时的探测间隔
	defaultInterval      = 5 * time.Second  // 兜底
)

var startSlackOnce sync.Once

func StartSlackDispatcher() {
	startSlackOnce.Do(func() {
		log.Println("✅ Slack 告警调度器启动（DB 控制模式）")
		go loopSlack()
	})
}

func loopSlack() {
	for {
		cfg, err := config.GetSlackConfigUI() // 返回 SlackConfigRow, error
		if err != nil {
			log.Printf("slack.dispatcher: 读取配置失败: %v", err)
			time.Sleep(disabledPollInterval)
			continue
		}

		if cfg.Enable == 0 || strings.TrimSpace(cfg.Webhook) == "" {
			time.Sleep(disabledPollInterval)
			continue
		}

		iv := time.Duration(cfg.IntervalSec) * time.Second
		if iv <= 0 {
			iv = defaultInterval
		}

		slack.DispatchSlackAlertFromCleanedEvents(cfg.Webhook)
		time.Sleep(iv)
	}
}
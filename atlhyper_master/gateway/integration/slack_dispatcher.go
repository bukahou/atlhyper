// gateway/integration/slack_dispatcher.go
package integration

import (
	"context"
	"log"
	"strings"
	"sync"
	"time"

	"AtlHyper/atlhyper_master/gateway/integration/slack"
	"AtlHyper/atlhyper_master/service/db/config"
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
	ctx := context.Background()
	for {
		cfg, err := config.GetSlackConfigUI(ctx)
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

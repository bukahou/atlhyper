package integration

import (
	"context"
	"log"
	"strings"
	"sync"
	"time"

	"AtlHyper/atlhyper_master/gateway/integration/mailer"
	"AtlHyper/atlhyper_master/service/db/config"
)

const (
	mailDisabledPollInterval = 30 * time.Second // 关闭/未配置时的探测间隔
	mailDefaultInterval      = 60 * time.Second // 兜底
)

var startMailOnce sync.Once

// StartEmailDispatcher 启动邮件告警调度器（DB 控制模式）
func StartEmailDispatcher() {
	startMailOnce.Do(func() {
		log.Println("✅ 邮件告警调度器启动（DB 控制模式）")
		go loopMail()
	})
}

func loopMail() {
	ctx := context.Background()
	for {
		cfg, err := config.GetMailConfigUI(ctx)
		if err != nil {
			log.Printf("mail.dispatcher: 读取配置失败: %v", err)
			time.Sleep(mailDisabledPollInterval)
			continue
		}

		// 检查是否启用以及必要配置是否完整
		if cfg.Enable == 0 ||
			strings.TrimSpace(cfg.SMTPHost) == "" ||
			strings.TrimSpace(cfg.FromAddr) == "" ||
			strings.TrimSpace(cfg.ToAddrs) == "" {
			time.Sleep(mailDisabledPollInterval)
			continue
		}

		iv := time.Duration(cfg.IntervalSec) * time.Second
		if iv <= 0 {
			iv = mailDefaultInterval
		}

		// 调用邮件发送（传入配置）
		mailer.DispatchEmailAlertFromCleanedEvents(mailer.MailConfig{
			SMTPHost: cfg.SMTPHost,
			SMTPPort: cfg.SMTPPort,
			Username: cfg.Username,
			Password: cfg.Password,
			From:     cfg.FromAddr,
			To:       strings.Split(cfg.ToAddrs, ","),
		})

		time.Sleep(iv)
	}
}

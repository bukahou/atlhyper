package alerter

import (
	"NeuroController/internal/mailer"
	"NeuroController/internal/utils"
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

var (
	lastEmailSentTimeMu sync.Mutex
	lastEmailSentTime   time.Time
)

const throttleInterval = 1 * time.Hour

// ✅ 外部统一调用这个节流版本，内部仍使用三参原始函数
func SendAlertEmailWithThrottle(to []string, subject string, data mailer.AlertGroupData, eventTime time.Time) error {
	lastEmailSentTimeMu.Lock()
	defer lastEmailSentTimeMu.Unlock()

	if !lastEmailSentTime.IsZero() && time.Since(lastEmailSentTime) < throttleInterval {
		utils.Info(context.TODO(), "⏳ 邮件告警触发，但处于节流期内，跳过发送",
			zap.Duration("剩余等待", throttleInterval-time.Since(lastEmailSentTime)),
		)
		return nil
	}

	// 更新时间戳并发送邮件
	lastEmailSentTime = time.Now()
	utils.Info(context.TODO(), "📨 满足条件，正在发送邮件",
		zap.String("subject", subject),
		zap.Time("sendTime", lastEmailSentTime),
	)
	return mailer.SendAlertEmail(to, subject, data)
}

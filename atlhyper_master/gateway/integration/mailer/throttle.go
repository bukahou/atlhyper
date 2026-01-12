package mailer

import (
	"AtlHyper/model/integration"
	"sync"
	"time"
)

// 节流控制机制 - 防止邮件频繁发送
var (
	lastEmailSentTimeMu sync.Mutex
	lastEmailSentTime   time.Time
)

// 节流时间间隔（1 小时）
const throttleInterval = 1 * time.Hour

// SendAlertEmailWithThrottle 节流判断后发送告警邮件
func SendAlertEmailWithThrottle(cfg MailConfig, subject string, data integration.AlertGroupData, eventTime time.Time) error {
	lastEmailSentTimeMu.Lock()
	defer lastEmailSentTimeMu.Unlock()

	// 若上次发送时间非零，且距离当前不足 throttleInterval，则跳过
	if !lastEmailSentTime.IsZero() && time.Since(lastEmailSentTime) < throttleInterval {
		return nil
	}

	// 满足节流条件：更新记录时间，并发送邮件
	lastEmailSentTime = time.Now()
	return SendAlertEmail(cfg, subject, data)
}

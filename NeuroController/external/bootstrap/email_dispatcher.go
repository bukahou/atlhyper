// =======================================================================================
// 📄 external/bootstrap/email_dispatcher.go
//
// 📬 Description:
//     核心邮件告警调度器。该模块由诊断系统周期性调用，统一处理清洗后的告警事件。
//     若事件满足触发条件，则构造邮件并通过节流控制器发送邮件告警。
//
// ⚙️ Responsibilities:
//     - 从 diagnosis 获取已清洗事件
//     - 调用 alerter 模块判断是否触发告警
//     - 构造 AlertGroupData 并通过 mailer 发送（支持节流）
//
// 📣 推荐由清理器模块周期性调度调用
//
// ✍️ Author: bukahou (@ZGMF-X10A)
// =======================================================================================

package bootstrap

import (
	"NeuroController/config"
	"NeuroController/external/mailer"
	"NeuroController/interfaces"
	"fmt"
	"time"
)

// ✅ 启动定时邮件告警调度器（推荐在控制器启动时调用）
//
// 行为：每隔 EmailInterval 周期性调用 DispatchEmailAlertFromCleanedEvents
func StartEmailDispatcher() {
	emailInterval := config.GlobalConfig.Diagnosis.AlertDispatchInterval

	// 启动提示日志
	fmt.Println("📬 启动邮件告警调度器 ...")
	fmt.Printf("⏱️ 告警检测周期：%v\n", emailInterval)

	// ✅ 启动异步循环
	go func() {
		for {
			DispatchEmailAlertFromCleanedEvents()
			time.Sleep(emailInterval)
		}
	}()

	fmt.Println("✅ 邮件调度器启动成功。")
}

func DispatchEmailAlertFromCleanedEvents() {
	events := interfaces.GetCleanedEventLogs()
	shouldAlert, subject, data := interfaces.ComposeAlertGroupIfNecessary(events)

	if shouldAlert {
		recipients := config.GlobalConfig.Mailer.To
		err := mailer.SendAlertEmailWithThrottle(recipients, subject, data, time.Now())
		if err != nil {
			fmt.Printf("❌ 邮件发送失败: %v\n", err)
		}
	}
}

// =======================================================================================
// 📄 interface/alert_group_evaluation_api.go
//
// 📦 Description:
//     聚合调用清洗事件池和告警构建逻辑，返回是否需要触发告警、邮件/Slack 标题和告警数据体。
//     封装 GetCleanedEventLogs 和 ComposeAlertGroupIfNecessary 的组合调用。
//
// 🔌 Responsibilities:
//     - 获取最新清洗事件
//     - 判断是否需要触发告警
//     - 返回结构化告警内容（供邮件/Slack/其他通知模块调用）
//
// ✍️ Author: bukahou (@ZGMF-X10A)
// =======================================================================================

package interfaces

import (
	"NeuroController/internal/types"
	"fmt"
)

// ✅ 聚合评估函数：用于对外获取当前是否应触发告警
func GetAlertGroupIfNecessary() (bool, string, types.AlertGroupData) {
	fmt.Println("🧠 [AlertEval] 开始评估清洗后的事件池...")

	events := GetCleanedEventLogs()
	if len(events) == 0 {
		fmt.Println("ℹ️ [AlertEval] 当前清洗事件池为空，无需评估告警。")
		return false, "", types.AlertGroupData{}
	}

	fmt.Printf("📦 [AlertEval] 共获取 %d 条清洗事件，开始格式化评估...\n", len(events))

	shouldAlert, subject, data := ComposeAlertGroupIfNecessary(events)
	if shouldAlert {
		fmt.Printf("📬 [AlertEval] 告警评估通过，生成 AlertGroupData，标题: \"%s\"，告警数: %d\n", subject, data.AlertCount)
	} else {
		fmt.Println("✅ [AlertEval] 当前无需触发告警，系统状态正常。")
	}

	return shouldAlert, subject, data
}

package testapi

import (
	"AtlHyper/atlhyper_master/client/alert"
	"net/http"

	"github.com/gin-gonic/gin"
)

//
// HandleRunAIDiagnosis —— 获取 AI 诊断所需的原始事件整合数据
// ------------------------------------------------------------
// 📍 Endpoint: GET /testapi/ai/diagnose/run
//
// ✅ 功能：
//   - 调用 alert.CollectNewEventsGroupedForAI() 收集并分组增量事件
//   - 原样返回结构体（每个 Cluster 一组）
//   - 方便手动发送到 aiservice 测试 AI 分析效果
//
// 🚫 不执行任何 AI 请求，只输出原始数据。
//
func HandleRunAIDiagnosis(c *gin.Context) {
	// 1️⃣ 获取整合结果
	groups := alert.CollectNewEventsGroupedForAI()

	// 2️⃣ 若无新事件
	if len(groups) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"message": "暂无可分析的新事件（agent 端暂无告警级上报）",
			"groups":  []alert.ClusterEventGroup{},
		})
		return
	}

	// 3️⃣ 统计总事件数
	totalEvents := 0
	for _, g := range groups {
		totalEvents += g.Count
	}

	// 4️⃣ 返回完整数据
	c.JSON(http.StatusOK, gin.H{
		"message":      "✅ 已生成 AI 分析所需原始数据",
		"clusterCount": len(groups),
		"eventCount":   totalEvents,
		"rawData":      groups, // 直接返回完整结构
	})
}

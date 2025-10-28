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
		})
		return
	}

	// 3️⃣ 如果只有一个集群 → 直接返回该集群的事件组结构（便于直接复制测试）
	if len(groups) == 1 {
		c.JSON(http.StatusOK, groups[0])
		return
	}

	// 4️⃣ 如果存在多个集群 → 提供全部（保留原始结构）
	c.JSON(http.StatusOK, gin.H{
		"message":      "✅ 检测到多个集群事件组，请手动选择需要测试的集群",
		"clusterCount": len(groups),
		"rawData":      groups,
	})
}

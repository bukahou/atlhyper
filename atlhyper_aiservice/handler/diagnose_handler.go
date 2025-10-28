// atlhyper_aiservice/handler/diagnose_handler.go
package handler

import (
	"AtlHyper/atlhyper_aiservice/service"
	m "AtlHyper/model/event" // ✅ 引入共用的事件模型（全项目统一结构）
	"net/http"

	"github.com/gin-gonic/gin"
)

//
// DiagnoseEventHandler —— 处理 Master 发送的事件诊断请求
// ------------------------------------------------------------
// 📍 Endpoint: POST /ai/diagnose
// ✅ 主要职责：
//   - 接收 Master 传入的集群事件列表（[]model.EventLog）
//   - 调用 AI 分析服务（service.DiagnoseEvents）
//   - 返回分析结果（包括 AI 的原始输出与摘要）
//
// 🔧 使用场景：
//   由 AtlHyper Master 端在检测到新的增量事件（CollectNewEventLogsForAlert）后，
//   自动通过 HTTP 调用本接口，将事件传递给 AI Service 进行初步诊断。
//
func DiagnoseEventHandler(c *gin.Context) {
	// 请求体结构：ClusterID + Events 列表
	var req struct {
		ClusterID string       `json:"clusterID"` // 集群唯一标识
		Events    []m.EventLog `json:"events"`    // 事件列表（使用统一结构 model.EventLog）
	}

	// 1️⃣ 校验请求格式与事件数量
	if err := c.ShouldBindJSON(&req); err != nil || len(req.Events) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid payload or empty events",
			"example": `{"clusterID":"cluster-1","events":[{"kind":"Pod","reason":"CrashLoopBackOff","message":"container restart"}]}`,
		})
		return
	}

	// 2️⃣ 调用 service 层进行 AI 分析
	resp, err := service.DiagnoseEvents(req.ClusterID, req.Events)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to analyze events",
			"details": err.Error(),
		})
		return
	}

	// 3️⃣ 成功返回结果
	c.JSON(http.StatusOK, gin.H{
		"clusterID": req.ClusterID, // 方便上层对应集群
		"response":  resp,          // 包含分析摘要、prompt、AI 原文输出
	})
}

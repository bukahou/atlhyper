// atlhyper_aiservice/handler/diagnose_handler.go
package handler

import (
	"AtlHyper/atlhyper_aiservice/service"
	m "AtlHyper/model/event" // ✅ 引入共用事件模型
	"net/http"

	"github.com/gin-gonic/gin"
)

//
// DiagnoseEventHandler —— 处理 Master 发送的事件诊断请求
// ------------------------------------------------------------
// 📍 Endpoint: POST /ai/diagnose
// ✅ 主要职责：
//   - 接收 Master 传入的集群事件列表（[]model.EventLog）
//   - 调用 AI 分析服务（service.RunAIDiagnosisPipeline）
//   - 返回分析结果（包括多阶段的 AI 输出）
//
// 🔧 使用场景：
//   由 AtlHyper Master 端在检测到新事件后，
//   通过 HTTP 调用本接口，将事件交由 AI Service 进行智能诊断。
//
func DiagnoseEventHandler(c *gin.Context) {
	// 1️⃣ 解析请求体
	var req struct {
		ClusterID string       `json:"clusterID"` // 集群唯一标识
		Events    []m.EventLog `json:"events"`    // 事件列表
	}
	if err := c.ShouldBindJSON(&req); err != nil || len(req.Events) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    40000,
			"message": "invalid payload or empty events",
			"example": `{"clusterID":"cluster-1","events":[{"kind":"Pod","reason":"CrashLoopBackOff","message":"container restart"}]}`,
		})
		return
	}

	// 2️⃣ 调用 service 层执行完整诊断流水线
	ctx := c.Request.Context()
	resp, err := service.RunAIDiagnosisPipeline(ctx, req.ClusterID, req.Events)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    50000,
			"message": "AI 诊断执行失败",
			"error":   err.Error(),
		})
		return
	}

	// 3️⃣ 成功返回结果
	c.JSON(http.StatusOK, gin.H{
		"code":      20000,
		"message":   "success",
		"clusterID": req.ClusterID,
		"data":      resp, // 包含 stage1, stage2, stage3
	})
}

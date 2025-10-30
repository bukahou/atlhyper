// atlhyper_aiservice/handler/diagnose_handler.go
package handler

import (
	"AtlHyper/atlhyper_aiservice/service/diagnose"
	m "AtlHyper/model/event"
	"net/http"

	"github.com/gin-gonic/gin"
)

// DiagnoseEventHandler —— 处理 Master 发送的事件诊断请求
// ------------------------------------------------------------
// 📍 Endpoint: POST /ai/diagnose
// ✅ 功能说明：
//   - 接收 Master 传入的事件列表（[]model.EventLog）
//   - 调用 AI 服务执行诊断流水线
//   - 返回诊断结果（含多阶段 AI 输出）
func DiagnoseEventHandler(c *gin.Context) {
	// 1️⃣ 解析请求体
	var req struct {
		ClusterID string       `json:"clusterID"`
		Events    []m.EventLog `json:"events"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || len(req.Events) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "请求体无效或事件列表为空",
			"example": `{"clusterID":"cluster-1","events":[{"kind":"Pod","reason":"CrashLoopBackOff","message":"容器重启"}]}`,
		})
		return
	}

	// 2️⃣ 执行诊断
	ctx := c.Request.Context()
	resp, err := diagnose.RunAIDiagnosisPipeline(ctx, req.ClusterID, req.Events)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "AI 诊断执行失败",
			"error":   err.Error(),
		})
		return
	}

	// 3️⃣ 返回结果
	c.JSON(http.StatusOK, gin.H{
		"message":   "诊断成功",
		"clusterID": req.ClusterID,
		"data":      resp,
	})
}

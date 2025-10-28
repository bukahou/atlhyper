package handler

import (
	"net/http"

	"AtlHyper/atlhyper_aiservice/service"

	"github.com/gin-gonic/gin"
)

// AiTestHandler 通用测试接口：GET /ai/test?prompt=xxx
func AiTestHandler(c *gin.Context) {
	prompt := c.Query("prompt")
	if prompt == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "missing prompt parameter",
			"example": "/ai/test?prompt=Hello+AI",
		})
		return
	}

	text, err := service.GenerateByGemini(prompt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to generate text",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"prompt":   prompt,
		"response": text,
	})
}

// GenerateOpsReportHandler 运维报告生成：POST /ai/report
// func GenerateOpsReportHandler(c *gin.Context) {
// 	var req struct {
// 		Summary string `json:"summary"`
// 	}
// 	if err := c.ShouldBindJSON(&req); err != nil || req.Summary == "" {
// 		c.JSON(http.StatusBadRequest, gin.H{
// 			"error":   "missing 'summary' in request body",
// 			"example": `{"summary": "今日K8s节点重启2次，服务恢复正常"}`,
// 		})
// 		return
// 	}

// 	text, err := service.GenerateOpsReport(req.Summary)
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{
// 			"error":   "failed to generate report",
// 			"details": err.Error(),
// 		})
// 		return
// 	}

// 	c.JSON(http.StatusOK, gin.H{
// 		"summary":  req.Summary,
// 		"response": text,
// 	})
// }

// AnalyzeLogHandler 日志分析接口：POST /ai/analyze
// func AnalyzeLogHandler(c *gin.Context) {
// 	var req struct {
// 		Logs string `json:"logs"`
// 	}
// 	if err := c.ShouldBindJSON(&req); err != nil || req.Logs == "" {
// 		c.JSON(http.StatusBadRequest, gin.H{
// 			"error":   "missing 'logs' in request body",
// 			"example": `{"logs": "Error: pod crashloopbackoff detected on node desk-one"}`,
// 		})
// 		return
// 	}

// 	text, err := service.AnalyzeLogPattern(req.Logs)
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{
// 			"error":   "failed to analyze logs",
// 			"details": err.Error(),
// 		})
// 		return
// 	}

// 	c.JSON(http.StatusOK, gin.H{
// 		"logs":     req.Logs,
// 		"response": text,
// 	})
// }

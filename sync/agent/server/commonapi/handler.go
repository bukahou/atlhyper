package commonapi

import (
	"NeuroController/interfaces"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GET /sync/commonapi/cleaned-events
func HandleCleanedEvents(c *gin.Context) {
	// 💡 当前 GetCleanedEvents 没有 error 返回
	events := interfaces.GetCleanedEventLogs()

	// 安全性检查（理论上不会为 nil，但为保险）
	if events == nil {
		log.Println("⚠️ 获取清理事件失败或为空")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "无法获取清理后的事件",
		})
		return
	}

		// ✅ 打印事件数量和部分内容（可视情况裁剪）
	log.Printf("✅ 返回清理后的事件，共 %d 条\n", len(events))

	// ✅ 正常返回
	c.JSON(http.StatusOK, events)
}

// GET /sync/commonapi/alert-group
func HandleAlertGroup(c *gin.Context) {
	// 获取清理后的异常事件
	events := interfaces.GetCleanedEventLogs()

	// 组装告警组（根据策略判断是否需要告警）
	shouldAlert, subject, data := interfaces.ComposeAlertGroupIfNecessary(events)

	if !shouldAlert {
		log.Println("✅ 当前不满足告警条件，无需发送")
		c.JSON(http.StatusOK, gin.H{
			"alert": false,
			"note":  "当前不满足告警条件",
		})
		return
	}

	// ✅ 满足告警条件，返回告警内容
	log.Printf("🚨 满足告警条件：%s，共 %d 条异常\n", subject, data.AlertCount)
	c.JSON(http.StatusOK, gin.H{
		"alert":  true,
		"title":  subject,
		"data":   data,
	})
}

// GET /sync/commonapi/alert-group-light
func HandleLightweightAlertGroup(c *gin.Context) {
	// 获取清理后的事件
	events := interfaces.GetCleanedEventLogs()

	// 生成轻量化告警数据
	shouldDisplay, title, data := interfaces.GetLightweightAlertGroup(events)

	if !shouldDisplay {
		log.Println("✅ 当前无告警事件（轻量模式）")
		c.JSON(http.StatusOK, gin.H{
			"display": false,
			"note":    "当前无活跃告警",
		})
		return
	}

	log.Printf("📋 返回轻量告警信息：%s，%d 条\n", title, data.AlertCount)

	c.JSON(http.StatusOK, gin.H{
		"display": true,
		"title":   title,
		"data":    data,
	})
}
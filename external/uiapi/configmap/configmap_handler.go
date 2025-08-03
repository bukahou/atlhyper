// =======================================================================================
// 📄 configmap_handler.go（external/uiapi/configmap）
//
// ✨ 文件说明：
//     提供 ConfigMap 资源的 HTTP 路由处理逻辑，连接 interfaces 层逻辑与外部请求。
//     实现功能包括：
//       - 查询所有命名空间下的 ConfigMap
//       - 查询指定命名空间下的 ConfigMap
//       - 获取指定 ConfigMap 的详情
//
// 📍 路由前缀：/uiapi/configmap/**
//
// 📦 依赖模块：
//     - interfaces/ui_api/configmap_api.go
//
// ✍️ 作者：bukahou (@ZGMF-X10A)
// =======================================================================================

package configmap

import (
	"NeuroController/external/uiapi/response"
	"NeuroController/sync/center/http/uiapi"
	"net/http"

	"github.com/gin-gonic/gin"
)

// =======================================================================================
// ✅ GET /uiapi/configmap/list/by-namespace/:ns
//
// 🔍 查询指定命名空间下的 ConfigMap 列表
// =======================================================================================
func ListConfigMapsByNamespaceHandler(c *gin.Context) {
	ns := c.Param("ns")

	list, err := uiapi.GetConfigMapsByNamespace(ns)
	if err != nil {
		response.Error(c, "获取 ConfigMap 列表失败: "+err.Error())
		return
	}
	response.Success(c, "获取成功", list)
}


// =======================================================================================
// ✅ GET /uiapi/configmap/get/:ns/:name
//
// 🔍 获取指定命名空间和名称的 ConfigMap 详情
// =======================================================================================
func GetConfigMapDetailHandler(c *gin.Context) {
	ns := c.Param("ns")
	name := c.Param("name")

	cfg, err := uiapi.GetConfigMapDetail(ns, name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取 ConfigMap 详情失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, cfg)
}

// =======================================================================================
// ✅ GET /uiapi/configmap/list
//
// 🔍 查询所有命名空间下的 ConfigMap 列表（用于全局视图）
// =======================================================================================
func ListAllConfigMapsHandler(c *gin.Context) {
	list, err := uiapi.GetAllConfigMaps()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取所有 ConfigMap 失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, list)
}



// =======================================================================================
// ✅ GET /uiapi/configmap/alert/get
//
// 🔍 获取当前告警系统的配置信息（ConfigMap 字段）
// =======================================================================================
func GetAlertSettingsHandler(c *gin.Context) {
	data, err := uiapi.GetAlertConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取告警配置失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}

// =======================================================================================
// ✅ POST /uiapi/configmap/alert/slack
//
// ✏️ 更新 Slack 配置（开关 + webhook 地址）
// Body: { "enabled": true, "webhook": "https://..." }
// =======================================================================================
func UpdateSlackConfigHandler(c *gin.Context) {
	var req struct {
		Enabled bool   `json:"enabled"`
		Webhook string `json:"webhook"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式错误: " + err.Error()})
		return
	}

	err := uiapi.UpdateSlack(req.Enabled, req.Webhook)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新 Slack 配置失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Slack 配置已更新"})
}

// =======================================================================================
// ✅ POST /uiapi/configmap/alert/webhook
//
// ✏️ 更新 Webhook 开关（CI/CD 更新）
// Body: { "enabled": true }
// =======================================================================================
func UpdateWebhookSwitchHandler(c *gin.Context) {
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式错误: " + err.Error()})
		return
	}

	err := uiapi.UpdateWebhook(req.Enabled)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新 Webhook 开关失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Webhook 开关已更新"})
}

// =======================================================================================
// ✅ POST /uiapi/configmap/alert/mail
//
// ✏️ 更新邮件配置（开关、用户名、密码、发件人、收件人）
// Body: {
//   "enabled": true,
//   "username": "xxx@gmail.com",
//   "password": "xxx",
//   "from": "xxx@gmail.com",
//   "to": ["a@x.com", "b@x.com"]
// }
// =======================================================================================
func UpdateMailConfigHandler(c *gin.Context) {
	var req struct {
		Enabled  bool     `json:"enabled"`
		Username string   `json:"username"`
		Password string   `json:"password"`
		From     string   `json:"from"`
		To       []string `json:"to"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式错误: " + err.Error()})
		return
	}

	err := uiapi.UpdateMail(req.Enabled, req.Username, req.Password, req.From, req.To)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新邮件配置失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "邮件配置已更新"})
}

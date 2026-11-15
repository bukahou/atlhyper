// atlhyper_master_v2/gateway/handler/admin/notify_smtp.go
// 通知设置 — 全局发件设置（邮件接收人共用的 SMTP 服务器与账号）
package admin

import (
	"net/http"

	"AtlHyper/atlhyper_master_v2/model"
)

// GetSMTP 当前的发件设置（不含密码，只说明是否已配置）
// GET /api/v2/notify/smtp
func (h *NotifyHandler) GetSMTP(w http.ResponseWriter, r *http.Request) {
	settings, err := h.querySvc.GetNotifySMTP(r.Context())
	if err != nil {
		h.writeError(w, "获取发件设置失败", err)
		return
	}
	writeNotifyData(w, http.StatusOK, "获取成功", settings)
}

// UpdateSMTP 保存发件设置（整体替换；password 留空 = 保持原值）
// PUT /api/v2/notify/smtp
func (h *NotifyHandler) UpdateSMTP(w http.ResponseWriter, r *http.Request) {
	var req model.UpdateNotifySMTPRequest
	if !decodeNotifyBody(w, r, &req) {
		return
	}
	if err := h.opsSvc.UpdateNotifySMTP(r.Context(), &req); err != nil {
		h.writeError(w, "保存发件设置失败", err)
		return
	}
	// 读回失败不改变结果：写入已经生效，只是不带 data
	settings, err := h.querySvc.GetNotifySMTP(r.Context())
	if err != nil {
		notifyLog.Warn("发件设置已保存，但读回失败", "err", err)
		writeNotifyData(w, http.StatusOK, "已保存", nil)
		return
	}
	writeNotifyData(w, http.StatusOK, "已保存", settings)
}

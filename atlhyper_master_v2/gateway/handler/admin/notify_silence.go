// atlhyper_master_v2/gateway/handler/admin/notify_silence.go
// 通知设置 Handler：静默（列表、新建、提前结束）
package admin

import (
	"context"
	"net/http"

	"AtlHyper/atlhyper_master_v2/gateway/handler"
	"AtlHyper/atlhyper_master_v2/gateway/middleware"
	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/atlhyper_master_v2/notifier/silence"
)

// ListSilences 静默列表
// GET /api/v2/notify/silences?state=active|pending|expired
func (h *NotifyHandler) ListSilences(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")
	if !oneOf(state, "", silence.StatePending, silence.StateActive, silence.StateExpired) {
		handler.WriteError(w, http.StatusBadRequest, "state 须为 active / pending / expired 之一")
		return
	}
	silences, err := h.querySvc.ListNotifySilences(r.Context(), state)
	if err != nil {
		h.writeError(w, "获取静默列表失败", err)
		return
	}
	if silences == nil {
		silences = []model.NotifySilence{}
	}
	writeNotifyList(w, silences, len(silences))
}

// CreateSilence 新建静默。创建人取自当前登录的主体，不看请求体
// POST /api/v2/notify/silences
func (h *NotifyHandler) CreateSilence(w http.ResponseWriter, r *http.Request) {
	var req model.CreateNotifySilenceRequest
	if !decodeNotifyBody(w, r, &req) {
		return
	}
	createdBy, _ := middleware.GetUsername(r.Context())
	id, err := h.opsSvc.CreateNotifySilence(r.Context(), &req, createdBy)
	if err != nil {
		h.writeError(w, "创建静默失败", err)
		return
	}
	writeNotifyData(w, http.StatusCreated, "已创建", h.currentSilence(r.Context(), id))
}

// ExpireSilence 提前结束静默（还没开始的也可以，用于撤销建错的静默）
// POST /api/v2/notify/silences/{id}/expire
func (h *NotifyHandler) ExpireSilence(w http.ResponseWriter, r *http.Request) {
	id, ok := notifyPathID(w, r)
	if !ok {
		return
	}
	if err := h.opsSvc.ExpireNotifySilence(r.Context(), id); err != nil {
		h.writeError(w, "结束静默失败", err)
		return
	}
	writeNotifyData(w, http.StatusOK, "已结束", h.currentSilence(r.Context(), id))
}

// currentSilence 写操作成功后读回该静默，随响应带回；读回失败只记日志（写入已生效）
func (h *NotifyHandler) currentSilence(ctx context.Context, id int64) any {
	item, err := h.querySvc.GetNotifySilence(ctx, id)
	if err != nil {
		notifyLog.Warn("写入成功，但读回静默失败", "id", id, "err", err)
		return nil
	}
	if item == nil {
		return nil
	}
	return item
}

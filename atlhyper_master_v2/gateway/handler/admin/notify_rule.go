// atlhyper_master_v2/gateway/handler/admin/notify_rule.go
// 通知设置 Handler：路由规则（列表、新建、修改、删除、重排、试匹配）
package admin

import (
	"context"
	"net/http"

	"AtlHyper/atlhyper_master_v2/model"
)

// ListRules 全部路由规则，按 position 从小到大
// GET /api/v2/notify/rules
func (h *NotifyHandler) ListRules(w http.ResponseWriter, r *http.Request) {
	rules, err := h.querySvc.ListNotifyRules(r.Context())
	if err != nil {
		h.writeError(w, "获取路由规则失败", err)
		return
	}
	if rules == nil {
		rules = []model.NotifyRule{}
	}
	writeNotifyList(w, rules, len(rules))
}

// CreateRule 新建路由规则（排在最后）
// POST /api/v2/notify/rules
func (h *NotifyHandler) CreateRule(w http.ResponseWriter, r *http.Request) {
	var req model.NotifyRuleRequest
	if !decodeNotifyBody(w, r, &req) {
		return
	}
	id, err := h.opsSvc.CreateNotifyRule(r.Context(), &req)
	if err != nil {
		h.writeError(w, "创建路由规则失败", err)
		return
	}
	writeNotifyData(w, http.StatusCreated, "已创建", h.currentRule(r.Context(), id))
}

// UpdateRule 修改路由规则（不改位置）
// PUT /api/v2/notify/rules/{id}
func (h *NotifyHandler) UpdateRule(w http.ResponseWriter, r *http.Request) {
	id, ok := notifyPathID(w, r)
	if !ok {
		return
	}
	var req model.NotifyRuleRequest
	if !decodeNotifyBody(w, r, &req) {
		return
	}
	if err := h.opsSvc.UpdateNotifyRule(r.Context(), id, &req); err != nil {
		h.writeError(w, "修改路由规则失败", err)
		return
	}
	writeNotifyData(w, http.StatusOK, "已更新", h.currentRule(r.Context(), id))
}

// DeleteRule 删除路由规则
// DELETE /api/v2/notify/rules/{id}
func (h *NotifyHandler) DeleteRule(w http.ResponseWriter, r *http.Request) {
	id, ok := notifyPathID(w, r)
	if !ok {
		return
	}
	if err := h.opsSvc.DeleteNotifyRule(r.Context(), id); err != nil {
		h.writeError(w, "删除路由规则失败", err)
		return
	}
	writeNotifyData(w, http.StatusOK, "已删除", nil)
}

// ReorderRules 重排路由规则：ids 必须恰好是全部规则的一个排列
// PUT /api/v2/notify/rules/order
func (h *NotifyHandler) ReorderRules(w http.ResponseWriter, r *http.Request) {
	var req model.NotifyRuleOrderRequest
	if !decodeNotifyBody(w, r, &req) {
		return
	}
	if err := h.opsSvc.ReorderNotifyRules(r.Context(), req.IDs); err != nil {
		h.writeError(w, "重排路由规则失败", err)
		return
	}
	// 带回重排后的列表，调用方不必再拉一次；读回失败只是不带 data
	rules, err := h.querySvc.ListNotifyRules(r.Context())
	if err != nil {
		notifyLog.Warn("规则已重排，但读回失败", "err", err)
		writeNotifyData(w, http.StatusOK, "已保存", nil)
		return
	}
	if rules == nil {
		rules = []model.NotifyRule{}
	}
	writeNotifyData(w, http.StatusOK, "已保存", rules)
}

// MatchRule 试匹配：这样一条告警会命中哪条规则、发给谁（只读，不产生告警）
// POST /api/v2/notify/rules/match
func (h *NotifyHandler) MatchRule(w http.ResponseWriter, r *http.Request) {
	var req model.NotifyRuleMatchRequest
	if !decodeNotifyBody(w, r, &req) {
		return
	}
	result, err := h.querySvc.MatchNotifyRule(r.Context(), &req)
	if err != nil {
		h.writeError(w, "试匹配失败", err)
		return
	}
	writeNotifyData(w, http.StatusOK, "匹配完成", result)
}

// currentRule 写操作成功后读回该规则，随响应带回；读回失败只记日志（写入已生效）
func (h *NotifyHandler) currentRule(ctx context.Context, id int64) any {
	rule, err := h.querySvc.GetNotifyRule(ctx, id)
	if err != nil {
		notifyLog.Warn("写入成功，但读回路由规则失败", "id", id, "err", err)
		return nil
	}
	if rule == nil {
		return nil
	}
	return rule
}

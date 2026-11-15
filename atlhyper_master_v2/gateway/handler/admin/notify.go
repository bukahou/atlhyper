// atlhyper_master_v2/gateway/handler/admin/notify.go
// 通知设置 Handler：公共部分（依赖、参数解析、错误映射）与策略、当前告警、送达记录三组接口
//
// 接收人、路由规则、静默分别在 notify_receiver.go / notify_rule.go / notify_silence.go。
// ⛔ 任何响应都不带接收人的凭据；错误响应不回显请求内容，内部错误只记日志
package admin

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/gateway/handler"
	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/common/logger"
)

var notifyLog = logger.Module("NotifyHandler")

const (
	// notifyIDPathParam 路由里的路径参数名（见 gateway/routes.go 的 {id}）
	notifyIDPathParam = "id"
	// maxNotifyBodyBytes 请求体的大小上限
	maxNotifyBodyBytes = 64 << 10
	// 列表接口的分页：默认每页条数与上限
	defaultNotifyPageSize = 50
	maxNotifyPageSize     = 200
)

// NotifyQuery / NotifyOps 本 handler 所需的最小 service 能力
type NotifyQuery interface {
	GetNotifyPolicy(ctx context.Context) (*model.NotifyPolicy, error)
	GetNotifySMTP(ctx context.Context) (*model.NotifySMTP, error)
	ListNotifyAlerts(ctx context.Context, opts model.NotifyAlertQuery) ([]model.NotifyAlert, int, error)
	ListNotifyDeliveries(ctx context.Context, opts model.NotifyDeliveryQuery) ([]model.NotifyDelivery, int, error)
	ListNotifyReceivers(ctx context.Context) ([]model.NotifyReceiver, error)
	GetNotifyReceiver(ctx context.Context, id int64) (*model.NotifyReceiver, error)
	ListNotifyRules(ctx context.Context) ([]model.NotifyRule, error)
	GetNotifyRule(ctx context.Context, id int64) (*model.NotifyRule, error)
	ListNotifySilences(ctx context.Context, state string) ([]model.NotifySilence, error)
	GetNotifySilence(ctx context.Context, id int64) (*model.NotifySilence, error)
	MatchNotifyRule(ctx context.Context, req *model.NotifyRuleMatchRequest) (*model.NotifyRuleMatchResult, error)
}

type NotifyOps interface {
	UpdateNotifyPolicy(ctx context.Context, req *model.UpdateNotifyPolicyRequest) error
	UpdateNotifySMTP(ctx context.Context, req *model.UpdateNotifySMTPRequest) error
	CreateNotifyReceiver(ctx context.Context, req *model.NotifyReceiverRequest) (int64, error)
	UpdateNotifyReceiver(ctx context.Context, id int64, req *model.NotifyReceiverRequest) error
	DeleteNotifyReceiver(ctx context.Context, id int64) error
	CreateNotifyRule(ctx context.Context, req *model.NotifyRuleRequest) (int64, error)
	UpdateNotifyRule(ctx context.Context, id int64, req *model.NotifyRuleRequest) error
	DeleteNotifyRule(ctx context.Context, id int64) error
	ReorderNotifyRules(ctx context.Context, ids []int64) error
	CreateNotifySilence(ctx context.Context, req *model.CreateNotifySilenceRequest, createdBy string) (int64, error)
	ExpireNotifySilence(ctx context.Context, id int64) error
}

// NotifyTester 给一个接收人发一条测试消息（由通知模块提供）。
// 返回的错误不含 Webhook 地址与密码
type NotifyTester interface {
	TestReceiver(ctx context.Context, receiverID int64) error
}

// NotifyHandler 通知设置 Handler
type NotifyHandler struct {
	querySvc NotifyQuery
	opsSvc   NotifyOps
	tester   NotifyTester // 可选，nil = 测试发送不可用
}

// NewNotifyHandler 创建 NotifyHandler
func NewNotifyHandler(querySvc NotifyQuery, opsSvc NotifyOps) *NotifyHandler {
	return &NotifyHandler{querySvc: querySvc, opsSvc: opsSvc}
}

// SetTester 注入测试发送能力
func (h *NotifyHandler) SetTester(t NotifyTester) { h.tester = t }

// ==================== 策略 ====================

// GetPolicy 当前的通知策略
// GET /api/v2/notify/policy
func (h *NotifyHandler) GetPolicy(w http.ResponseWriter, r *http.Request) {
	policy, err := h.querySvc.GetNotifyPolicy(r.Context())
	if err != nil {
		h.writeError(w, "获取通知策略失败", err)
		return
	}
	writeNotifyData(w, http.StatusOK, "获取成功", policy)
}

// UpdatePolicy 保存通知策略（整体替换）
// PUT /api/v2/notify/policy
func (h *NotifyHandler) UpdatePolicy(w http.ResponseWriter, r *http.Request) {
	var req model.UpdateNotifyPolicyRequest
	if !decodeNotifyBody(w, r, &req) {
		return
	}
	if err := h.opsSvc.UpdateNotifyPolicy(r.Context(), &req); err != nil {
		h.writeError(w, "保存通知策略失败", err)
		return
	}
	// 读回失败不改变结果：写入已经生效，只是不带 data
	policy, err := h.querySvc.GetNotifyPolicy(r.Context())
	if err != nil {
		notifyLog.Warn("通知策略已保存，但读回失败", "err", err)
		writeNotifyData(w, http.StatusOK, "已保存", nil)
		return
	}
	writeNotifyData(w, http.StatusOK, "已保存", policy)
}

// ==================== 当前告警与送达记录 ====================

// ListAlerts 告警列表
// GET /api/v2/notify/alerts?status=firing|resolved&cluster_id=&limit=&offset=
func (h *NotifyHandler) ListAlerts(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	status := query.Get("status")
	if !oneOf(status, "", "firing", "resolved") {
		handler.WriteError(w, http.StatusBadRequest, "status 须为 firing 或 resolved")
		return
	}
	limit, offset, ok := parseNotifyPage(w, r)
	if !ok {
		return
	}
	alerts, total, err := h.querySvc.ListNotifyAlerts(r.Context(), model.NotifyAlertQuery{
		Status: status, ClusterID: handler.ClusterIDFromQuery(r), Limit: limit, Offset: offset,
	})
	if err != nil {
		h.writeError(w, "获取告警列表失败", err)
		return
	}
	if alerts == nil {
		alerts = []model.NotifyAlert{}
	}
	writeNotifyList(w, alerts, total)
}

// ListDeliveries 送达记录
// GET /api/v2/notify/deliveries?fingerprint=&status=&limit=&offset=
func (h *NotifyHandler) ListDeliveries(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	status := query.Get("status")
	if !oneOf(status, "", database.NotifyDeliveryPending, database.NotifyDeliverySuccess, database.NotifyDeliveryFailed,
		database.NotifyDeliveryAbandoned, database.NotifyDeliveryNoReceiver) {
		handler.WriteError(w, http.StatusBadRequest, "status 须为 pending / success / failed / abandoned / no-receiver 之一")
		return
	}
	limit, offset, ok := parseNotifyPage(w, r)
	if !ok {
		return
	}
	deliveries, total, err := h.querySvc.ListNotifyDeliveries(r.Context(), model.NotifyDeliveryQuery{
		Fingerprint: query.Get("fingerprint"), Status: status, Limit: limit, Offset: offset,
	})
	if err != nil {
		h.writeError(w, "获取送达记录失败", err)
		return
	}
	if deliveries == nil {
		deliveries = []model.NotifyDelivery{}
	}
	writeNotifyList(w, deliveries, total)
}

// ==================== 公共辅助 ====================

// writeNotifyData 统一的成功响应；data 为 nil 时不带 data
func writeNotifyData(w http.ResponseWriter, status int, message string, data any) {
	response := map[string]any{"message": message}
	if data != nil {
		response["data"] = data
	}
	handler.WriteJSON(w, status, response)
}

// writeNotifyList 列表响应：data 恒为数组，total 恒为数字
func writeNotifyList(w http.ResponseWriter, data any, total int) {
	handler.WriteJSON(w, http.StatusOK, map[string]any{"message": "获取成功", "data": data, "total": total})
}

// decodeNotifyBody 解析请求体（有大小上限）；失败时已写出 400，返回 false
func decodeNotifyBody(w http.ResponseWriter, r *http.Request, target any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxNotifyBodyBytes)
	if err := json.NewDecoder(r.Body).Decode(target); err != nil {
		handler.WriteError(w, http.StatusBadRequest, "请求参数无效")
		return false
	}
	return true
}

// notifyPathID 取路径里的 id；不是正整数时已写出 400，返回 false
func notifyPathID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue(notifyIDPathParam), 10, 64)
	if err != nil || id <= 0 {
		handler.WriteError(w, http.StatusBadRequest, "id 无效")
		return 0, false
	}
	return id, true
}

// parseNotifyPage 解析 limit / offset：没传 limit 用默认值，超过上限按上限；不是非负整数时已写出 400
func parseNotifyPage(w http.ResponseWriter, r *http.Request) (limit, offset int, ok bool) {
	limit = defaultNotifyPageSize
	query := r.URL.Query()
	if raw := query.Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			handler.WriteError(w, http.StatusBadRequest, "limit 须为正整数")
			return 0, 0, false
		}
		limit = min(parsed, maxNotifyPageSize)
	}
	if raw := query.Get("offset"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 0 {
			handler.WriteError(w, http.StatusBadRequest, "offset 须为非负整数")
			return 0, 0, false
		}
		offset = parsed
	}
	return limit, offset, true
}

func oneOf(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}

// writeError 把 service 返回的错误映射为状态码。
//
// 校验类错误与「被引用」的文案是写给管理员看的（它们不含凭据），原样返回；
// 其余错误只记日志，不向外透露内部细节
func (h *NotifyHandler) writeError(w http.ResponseWriter, action string, err error) {
	var inUse *model.NotifyReceiverInUseError
	switch {
	case errors.Is(err, model.ErrInvalidNotifyInput):
		handler.WriteError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, database.ErrNotifyRuleOrderMismatch):
		handler.WriteError(w, http.StatusBadRequest, "ids 必须恰好包含全部规则各一次")
	case errors.As(err, &inUse):
		handler.WriteError(w, http.StatusConflict, inUse.Error())
	case errors.Is(err, database.ErrNotifyReceiverNameTaken):
		handler.WriteError(w, http.StatusConflict, "接收人名称已存在")
	case errors.Is(err, database.ErrNotifyReceiverNotFound):
		handler.WriteError(w, http.StatusNotFound, "接收人不存在")
	case errors.Is(err, database.ErrNotifyRuleNotFound):
		handler.WriteError(w, http.StatusNotFound, "路由规则不存在")
	case errors.Is(err, database.ErrNotifySilenceNotFound):
		handler.WriteError(w, http.StatusNotFound, "静默不存在")
	default:
		// 这里的 err 来自库或加密层，不含请求里的凭据
		notifyLog.Error(action, "err", err)
		handler.WriteError(w, http.StatusInternalServerError, action)
	}
}

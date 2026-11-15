// atlhyper_master_v2/gateway/handler/admin/notify_receiver.go
// 通知设置 Handler：接收人（列表、新建、修改、删除、测试发送）
package admin

import (
	"context"
	"errors"
	"net/http"
	"time"

	"AtlHyper/atlhyper_master_v2/gateway/handler"
	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/atlhyper_master_v2/notifier"
)

// receiverTestTimeout 一次测试发送的超时
const receiverTestTimeout = 40 * time.Second

// ListReceivers 全部接收人（凭据只显示「是否已配置」）
// GET /api/v2/notify/receivers
func (h *NotifyHandler) ListReceivers(w http.ResponseWriter, r *http.Request) {
	receivers, err := h.querySvc.ListNotifyReceivers(r.Context())
	if err != nil {
		h.writeError(w, "获取接收人列表失败", err)
		return
	}
	if receivers == nil {
		receivers = []model.NotifyReceiver{}
	}
	writeNotifyList(w, receivers, len(receivers))
}

// CreateReceiver 新建接收人
// POST /api/v2/notify/receivers
func (h *NotifyHandler) CreateReceiver(w http.ResponseWriter, r *http.Request) {
	var req model.NotifyReceiverRequest
	if !decodeNotifyBody(w, r, &req) {
		return
	}
	id, err := h.opsSvc.CreateNotifyReceiver(r.Context(), &req)
	if err != nil {
		h.writeError(w, "创建接收人失败", err)
		return
	}
	writeNotifyData(w, http.StatusCreated, "已创建", h.currentReceiver(r.Context(), id))
}

// UpdateReceiver 修改接收人。凭据留空 = 保持原值
// PUT /api/v2/notify/receivers/{id}
func (h *NotifyHandler) UpdateReceiver(w http.ResponseWriter, r *http.Request) {
	id, ok := notifyPathID(w, r)
	if !ok {
		return
	}
	var req model.NotifyReceiverRequest
	if !decodeNotifyBody(w, r, &req) {
		return
	}
	if err := h.opsSvc.UpdateNotifyReceiver(r.Context(), id, &req); err != nil {
		h.writeError(w, "修改接收人失败", err)
		return
	}
	writeNotifyData(w, http.StatusOK, "已更新", h.currentReceiver(r.Context(), id))
}

// DeleteReceiver 删除接收人。仍被规则或默认策略引用时 409，写明被谁引用
// DELETE /api/v2/notify/receivers/{id}
func (h *NotifyHandler) DeleteReceiver(w http.ResponseWriter, r *http.Request) {
	id, ok := notifyPathID(w, r)
	if !ok {
		return
	}
	if err := h.opsSvc.DeleteNotifyReceiver(r.Context(), id); err != nil {
		h.writeError(w, "删除接收人失败", err)
		return
	}
	writeNotifyData(w, http.StatusOK, "已删除", nil)
}

// TestReceiver 按已保存的配置给接收人发一条测试消息
// POST /api/v2/notify/receivers/{id}/test
func (h *NotifyHandler) TestReceiver(w http.ResponseWriter, r *http.Request) {
	id, ok := notifyPathID(w, r)
	if !ok {
		return
	}
	if h.tester == nil {
		handler.WriteError(w, http.StatusServiceUnavailable, "通知模块未启用")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), receiverTestTimeout)
	defer cancel()

	err := h.tester.TestReceiver(ctx, id)
	switch {
	case err == nil:
		writeNotifyData(w, http.StatusOK, "测试消息已发送", nil)
	case errors.Is(err, notifier.ErrReceiverNotFound):
		handler.WriteError(w, http.StatusNotFound, "接收人不存在")
	case errors.Is(err, notifier.ErrReceiverConfigIncomplete):
		// 文案由通知模块用固定文字拼成（缺 Webhook 地址 / 收件人 / 发件设置），不含配置内容
		handler.WriteError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, notifier.ErrUnsupportedReceiverType):
		handler.WriteError(w, http.StatusBadRequest, "该接收人类型暂不支持发送")
	default:
		// 通知模块保证发送失败的原因里没有 Webhook 地址与密码，可以展示给管理员
		handler.WriteError(w, http.StatusBadGateway, "发送失败: "+err.Error())
	}
}

// currentReceiver 写操作成功后读回该接收人，随响应带回；读回失败只记日志（写入已生效）
func (h *NotifyHandler) currentReceiver(ctx context.Context, id int64) any {
	receiver, err := h.querySvc.GetNotifyReceiver(ctx, id)
	if err != nil {
		notifyLog.Warn("写入成功，但读回接收人失败", "id", id, "err", err)
		return nil
	}
	if receiver == nil {
		return nil
	}
	return receiver
}

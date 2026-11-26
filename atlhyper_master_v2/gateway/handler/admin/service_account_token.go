// atlhyper_master_v2/gateway/handler/admin/service_account_token.go
// 服务账号管理 Handler：列表、新建、修改、删除，签发与吊销访问令牌（AtlHyper 的服务账号，不是 K8s 的 ServiceAccount）
//
// ⛔ 令牌明文只出现在签发的那一次响应里（不缓存）；错误响应不回显请求内容，内部错误只记日志
package admin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/gateway/handler"
	"AtlHyper/atlhyper_master_v2/gateway/middleware"
	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/common/logger"
)

var serviceAccountLog = logger.Module("ServiceAccountHandler")

const (
	// 路由里的路径参数名（见 gateway/routes.go 的 {id} 与 {tokenId}）
	serviceAccountIDPathParam = "id"
	apiTokenIDPathParam       = "tokenId"
	// maxServiceAccountBodyBytes 请求体上限：三个接口的请求体都只有几个短字段
	maxServiceAccountBodyBytes = 16 << 10
)

// ServiceAccountQuery / ServiceAccountOps 本 handler 所需的最小 service 能力
type ServiceAccountQuery interface {
	ListServiceAccounts(ctx context.Context) ([]model.ServiceAccountView, error)
	GetServiceAccount(ctx context.Context, id int64) (*model.ServiceAccountView, error)
}

type ServiceAccountOps interface {
	CreateServiceAccount(ctx context.Context, req *model.CreateServiceAccountRequest) (int64, error)
	UpdateServiceAccount(ctx context.Context, id int64, req *model.UpdateServiceAccountRequest) error
	DeleteServiceAccount(ctx context.Context, id int64) error
	IssueAPIToken(ctx context.Context, accountID int64, req *model.IssueAPITokenRequest, issuedBy string) (*model.APITokenIssued, error)
	RevokeAPIToken(ctx context.Context, accountID, tokenID int64) error
}

// ServiceAccountHandler 服务账号管理 Handler
type ServiceAccountHandler struct {
	querySvc ServiceAccountQuery
	opsSvc   ServiceAccountOps
}

// NewServiceAccountHandler 创建 ServiceAccountHandler
func NewServiceAccountHandler(querySvc ServiceAccountQuery, opsSvc ServiceAccountOps) *ServiceAccountHandler {
	return &ServiceAccountHandler{querySvc: querySvc, opsSvc: opsSvc}
}

// List 全部服务账号（各带令牌元数据）与业务参数
// GET /api/v2/admin/service-accounts
func (h *ServiceAccountHandler) List(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	accounts, err := h.querySvc.ListServiceAccounts(r.Context())
	if err != nil {
		serviceAccountLog.Error("获取服务账号列表失败", "err", err)
		handler.WriteError(w, http.StatusInternalServerError, "获取服务账号列表失败")
		return
	}
	if accounts == nil {
		accounts = []model.ServiceAccountView{}
	}
	handler.WriteJSON(w, http.StatusOK, map[string]any{
		"message": "获取成功",
		"data":    accounts,
		"total":   len(accounts),
		"limits":  model.CurrentServiceAccountLimits(),
	})
}

// Create 新建服务账号
// POST /api/v2/admin/service-accounts
func (h *ServiceAccountHandler) Create(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	var req model.CreateServiceAccountRequest
	if !decodeServiceAccountBody(w, r, &req) {
		return
	}
	id, err := h.opsSvc.CreateServiceAccount(r.Context(), &req)
	if err != nil {
		writeServiceAccountError(w, "新建服务账号失败", err)
		return
	}
	handler.WriteJSON(w, http.StatusCreated, h.withCurrentRow(r.Context(), "已新建", id))
}

// Update 修改显示名、等级与启停（整体替换）
// PUT /api/v2/admin/service-accounts/{id}
func (h *ServiceAccountHandler) Update(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPut) {
		return
	}
	id, ok := pathID(w, r, serviceAccountIDPathParam)
	if !ok {
		return
	}
	var req model.UpdateServiceAccountRequest
	if !decodeServiceAccountBody(w, r, &req) {
		return
	}
	if err := h.opsSvc.UpdateServiceAccount(r.Context(), id, &req); err != nil {
		writeServiceAccountError(w, "修改服务账号失败", err)
		return
	}
	handler.WriteJSON(w, http.StatusOK, h.withCurrentRow(r.Context(), "已更新", id))
}

// Delete 删除服务账号（令牌随之删除）
// DELETE /api/v2/admin/service-accounts/{id}
func (h *ServiceAccountHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodDelete) {
		return
	}
	id, ok := pathID(w, r, serviceAccountIDPathParam)
	if !ok {
		return
	}
	if err := h.opsSvc.DeleteServiceAccount(r.Context(), id); err != nil {
		writeServiceAccountError(w, "删除服务账号失败", err)
		return
	}
	handler.WriteJSON(w, http.StatusOK, map[string]any{"message": "已删除"})
}

// IssueToken 签发访问令牌。明文只在这一次响应里出现
// POST /api/v2/admin/service-accounts/{id}/tokens
func (h *ServiceAccountHandler) IssueToken(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	id, ok := pathID(w, r, serviceAccountIDPathParam)
	if !ok {
		return
	}
	var req model.IssueAPITokenRequest
	if !decodeServiceAccountBody(w, r, &req) {
		return
	}
	issuedBy, _ := middleware.GetUsername(r.Context())
	issued, err := h.opsSvc.IssueAPIToken(r.Context(), id, &req, issuedBy)
	if err != nil {
		writeServiceAccountError(w, "签发访问令牌失败", err)
		return
	}
	// 响应体里有令牌明文：不允许浏览器或中间代理缓存
	w.Header().Set("Cache-Control", "no-store")
	handler.WriteJSON(w, http.StatusCreated, map[string]any{"message": "已签发", "data": issued})
}

// RevokeToken 吊销访问令牌（行保留，立即失效）
// DELETE /api/v2/admin/service-accounts/{id}/tokens/{tokenId}
func (h *ServiceAccountHandler) RevokeToken(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodDelete) {
		return
	}
	accountID, ok := pathID(w, r, serviceAccountIDPathParam)
	if !ok {
		return
	}
	tokenID, ok := pathID(w, r, apiTokenIDPathParam)
	if !ok {
		return
	}
	if err := h.opsSvc.RevokeAPIToken(r.Context(), accountID, tokenID); err != nil {
		writeServiceAccountError(w, "吊销访问令牌失败", err)
		return
	}
	handler.WriteJSON(w, http.StatusOK, map[string]any{"message": "已吊销"})
}

// withCurrentRow 写操作成功后的响应：带上该账号最新的那一行，前端不必再拉一次列表。
//
// 读回失败不改变结果 —— 写入已经生效，这时报错会让调用方误以为没写进去；只是不带 data。
func (h *ServiceAccountHandler) withCurrentRow(ctx context.Context, message string, id int64) map[string]any {
	response := map[string]any{"message": message}
	view, err := h.querySvc.GetServiceAccount(ctx, id)
	if err != nil {
		serviceAccountLog.Warn("写入成功，但读回服务账号失败", "account", id, "err", err)
		return response
	}
	if view != nil {
		response["data"] = view
	}
	return response
}

// allowMethod 处理函数只接受自己的方法（路由层已按方法注册，这里是第二道）；不符时已写出 405
func allowMethod(w http.ResponseWriter, r *http.Request, method string) bool {
	if r.Method != method {
		handler.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return false
	}
	return true
}

// pathID 取路径参数里的 id；不是正整数时已写出 400，返回 false
func pathID(w http.ResponseWriter, r *http.Request, name string) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue(name), 10, 64)
	if err != nil || id <= 0 {
		handler.WriteError(w, http.StatusBadRequest, "路径参数无效")
		return 0, false
	}
	return id, true
}

// decodeServiceAccountBody 解析请求体（有大小上限，规格里没有的字段一律拒绝）；失败时已写出 400
func decodeServiceAccountBody(w http.ResponseWriter, r *http.Request, target any) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxServiceAccountBodyBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		handler.WriteError(w, http.StatusBadRequest, "请求参数无效")
		return false
	}
	return true
}

// writeServiceAccountError 把 service 返回的错误映射为状态码（按哨兵错误判断，不看文字）。
// 校验类错误的文案是写给管理员看的，原样返回；其余错误只记日志，不向外透露内部细节
func writeServiceAccountError(w http.ResponseWriter, action string, err error) {
	switch {
	case errors.Is(err, model.ErrInvalidServiceAccountInput):
		handler.WriteError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, database.ErrUsernameTaken):
		handler.WriteError(w, http.StatusConflict, "账号名已被占用")
	case errors.Is(err, database.ErrAPITokenLimitReached):
		handler.WriteError(w, http.StatusConflict,
			fmt.Sprintf("该账号未吊销的令牌已达上限（%d 枚），请先吊销不用的令牌", model.MaxActiveAPITokensPerAccount))
	case errors.Is(err, database.ErrServiceAccountNotFound):
		handler.WriteError(w, http.StatusNotFound, "服务账号不存在")
	case errors.Is(err, database.ErrAPITokenNotFound):
		handler.WriteError(w, http.StatusNotFound, "访问令牌不存在")
	default:
		serviceAccountLog.Error(action, "err", err)
		handler.WriteError(w, http.StatusInternalServerError, action)
	}
}

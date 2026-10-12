// atlhyper_master_v2/gateway/handler/auth_me.go
// 当前主体与权限（前端唯一的权限来源）
package handler

import (
	"context"
	"net/http"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/gateway/middleware"
)

// AuthMeUser 当前登录用户
type AuthMeUser struct {
	ID          int64  `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
	Email       string `json:"email"`
	Role        int    `json:"role"`
}

// AuthMeResponse 当前主体：访客与身份无效者 user = null、level = 0
type AuthMeResponse struct {
	User        *AuthMeUser `json:"user"`
	Level       int         `json:"level"`
	Permissions []string    `json:"permissions"`
}

// AuthMeUserReader 读取用户资料
type AuthMeUserReader interface {
	GetUserByID(ctx context.Context, id int64) (*database.User, error)
}

// AuthMeHandler 当前主体处理器
type AuthMeHandler struct {
	users AuthMeUserReader
}

// NewAuthMeHandler 创建 AuthMeHandler
func NewAuthMeHandler(users AuthMeUserReader) *AuthMeHandler {
	return &AuthMeHandler{users: users}
}

// Get 返回当前主体与权限列表
// GET /api/v2/auth/me
func (h *AuthMeHandler) Get(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	p := middleware.GetPrincipal(r.Context())
	resp := AuthMeResponse{Level: p.Level, Permissions: permissionNames(p.Level)}

	if !p.IsGuest() {
		// 主体已由 Authenticate 校验过；这里只补展示用的资料
		user, err := h.users.GetUserByID(r.Context(), p.UserID)
		if err == nil && user != nil {
			resp.User = &AuthMeUser{
				ID:          user.ID,
				Username:    user.Username,
				DisplayName: user.DisplayName,
				Email:       user.Email,
				Role:        user.Role,
			}
		}
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message": "获取成功",
		"data":    resp,
	})
}

func permissionNames(level int) []string {
	perms := middleware.PermissionsFor(level)
	names := make([]string, len(perms))
	for i, p := range perms {
		names[i] = string(p)
	}
	return names
}

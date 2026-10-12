// atlhyper_master_v2/service/query/user.go
// QueryUser 实现 — 用户读取
//
// 2026-08-29：此前 admin/user.go 直接持有 database.UserRepository，
// 绕过 service 层（违反 CLAUDE.md「Gateway 直接访问 Database」禁令）。
package query

import (
	"context"

	"AtlHyper/atlhyper_master_v2/database"
)

func (q *QueryService) GetUserByID(ctx context.Context, id int64) (*database.User, error) {
	return q.userRepo.GetByID(ctx, id)
}

func (q *QueryService) GetUserByUsername(ctx context.Context, username string) (*database.User, error) {
	return q.userRepo.GetByUsername(ctx, username)
}

func (q *QueryService) ListUsers(ctx context.Context) ([]*database.User, error) {
	return q.userRepo.List(ctx)
}

// GetUserByUsernameForPasswordLogin 密码登录专用：外部身份账号视为不存在
func (q *QueryService) GetUserByUsernameForPasswordLogin(ctx context.Context, username string) (*database.User, error) {
	return q.userRepo.GetByUsernameForPasswordLogin(ctx, username)
}

// FindUserIDByIdentity 按外部身份（issuer + subject）查本地用户 ID
func (q *QueryService) FindUserIDByIdentity(ctx context.Context, issuer, subject string) (int64, bool, error) {
	return q.identityRepo.FindUserID(ctx, issuer, subject)
}

// ListFederatedUserIDs 拥有外部身份的用户 ID 集合（用户列表标记用）
func (q *QueryService) ListFederatedUserIDs(ctx context.Context) (map[int64]bool, error) {
	return q.identityRepo.ListFederatedUserIDs(ctx)
}

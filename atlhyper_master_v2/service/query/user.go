// atlhyper_master_v2/service/query/user.go
// QueryUser 实现 — 用户读取
//
// admin/user.go 的读取都经这里，Gateway 不直接访问数据库。
package query

import (
	"context"

	"AtlHyper/atlhyper_master_v2/database"
)

func (q *UserQuery) GetUserByID(ctx context.Context, id int64) (*database.User, error) {
	return q.userRepo.GetByID(ctx, id)
}

func (q *UserQuery) GetUserByUsername(ctx context.Context, username string) (*database.User, error) {
	return q.userRepo.GetByUsername(ctx, username)
}

// ListUsers 人的账号（用户管理只管人；服务账号另见 ListServiceAccounts）
func (q *UserQuery) ListUsers(ctx context.Context) ([]*database.User, error) {
	return q.userRepo.ListHumans(ctx)
}

// GetUserByUsernameForPasswordLogin 密码登录专用：外部身份账号与服务账号视为不存在
func (q *UserQuery) GetUserByUsernameForPasswordLogin(ctx context.Context, username string) (*database.User, error) {
	return q.userRepo.GetByUsernameForPasswordLogin(ctx, username)
}

// FindUserIDByIdentity 按外部身份（issuer + subject）查本地用户 ID
func (q *UserQuery) FindUserIDByIdentity(ctx context.Context, issuer, subject string) (int64, bool, error) {
	return q.identityRepo.FindUserID(ctx, issuer, subject)
}

// ListFederatedUserIDs 拥有外部身份的用户 ID 集合（用户列表标记用）
func (q *UserQuery) ListFederatedUserIDs(ctx context.Context) (map[int64]bool, error) {
	return q.identityRepo.ListFederatedUserIDs(ctx)
}

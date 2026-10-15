// atlhyper_master_v2/auth/oidc/accounts.go
// 上游身份 → 本地用户（查或建）
package oidc

import (
	"context"
	"errors"
	"fmt"

	"AtlHyper/atlhyper_master_v2/database"

	"github.com/bukahou/akasha/pkg/oidcrp"
)

var (
	errDisabled = errors.New("账号已被禁用")
	errRejected = errors.New("准入策略拒绝")
)

// maxUsernameAttempts 用户名冲突时最多尝试的候选数
const maxUsernameAttempts = 20

type accountResolver struct {
	accounts    Accounts
	admission   AdmissionPolicy
	issuer      string
	newUserRole int
}

// resolve 已有身份 → 对应账号（同步展示资料）；新身份 → 准入后建号
//
// ⛔ 邮箱只作展示：不按邮箱匹配已有账号（那等于把账号安全委托给上游的邮箱验证）
func (r *accountResolver) resolve(ctx context.Context, id oidcrp.Identity) (*database.User, error) {
	if user, found, err := r.existing(ctx, id); err != nil || found {
		return user, err
	}
	if !r.admission.Admit(id) {
		return nil, errRejected
	}

	base := baseUsername(id)
	for n := 0; n < maxUsernameAttempts; n++ {
		user := &database.User{
			Username:    usernameCandidate(base, n),
			DisplayName: id.Name,
			Email:       id.Email,
			Role:        r.newUserRole,
			Status:      1,
			// PasswordHash 留空：该账号对密码登录不可见（查询层排除外部身份账号）
		}
		err := r.accounts.CreateFederatedUser(ctx, user, r.issuer, id.Subject)
		switch {
		case err == nil:
			return user, nil
		case errors.Is(err, database.ErrUsernameTaken):
			continue
		case errors.Is(err, database.ErrIdentityExists):
			// 并发首登：另一个请求已建好同一身份 → 回到那个账号
			user, found, err := r.existing(ctx, id)
			if err == nil && !found {
				err = fmt.Errorf("身份冲突后未找到账号: %s", id.Subject)
			}
			return user, err
		default:
			return nil, fmt.Errorf("建号失败: %w", err)
		}
	}
	return nil, fmt.Errorf("用户名候选耗尽: %s", base)
}

// existing 查已关联的账号；禁用 → errDisabled；展示资料变化时同步
func (r *accountResolver) existing(ctx context.Context, id oidcrp.Identity) (*database.User, bool, error) {
	userID, found, err := r.accounts.FindUserIDByIdentity(ctx, r.issuer, id.Subject)
	if err != nil || !found {
		return nil, false, err
	}
	user, err := r.accounts.GetUserByID(ctx, userID)
	if err != nil {
		return nil, false, err
	}
	if user == nil {
		return nil, false, fmt.Errorf("身份指向的用户不存在: %d", userID)
	}
	if user.Status != 1 {
		return nil, true, errDisabled
	}
	if user.DisplayName != id.Name || user.Email != id.Email {
		user.DisplayName, user.Email = id.Name, id.Email
		if err := r.accounts.UpdateUser(ctx, user); err != nil {
			return nil, false, err
		}
	}
	return user, true, nil
}

// atlhyper_master_v2/service/operations/user.go
// OpsUser 实现 — 用户写入
//
// 2026-08-29：此前 admin/user.go 直接持有 database.UserRepository，
// 绕过 service 层（违反 CLAUDE.md「Gateway 直接访问 Database」禁令）。
// 密码哈希与权限校验仍在 handler，此处只做数据存取。
package operations

import (
	"context"

	"AtlHyper/atlhyper_master_v2/database"
)

// UserService 用户写入操作
type UserService struct {
	userRepo     database.UserRepository
	identityRepo database.UserIdentityRepository
}

// NewUserService 创建 UserService
func NewUserService(userRepo database.UserRepository, identityRepo database.UserIdentityRepository) *UserService {
	return &UserService{userRepo: userRepo, identityRepo: identityRepo}
}

func (s *UserService) CreateUser(ctx context.Context, user *database.User) error {
	return s.userRepo.Create(ctx, user)
}

func (s *UserService) UpdateUser(ctx context.Context, user *database.User) error {
	return s.userRepo.Update(ctx, user)
}

// DeleteUser 删除用户，并清掉其外部身份行（否则同一身份再登录会撞唯一键）
func (s *UserService) DeleteUser(ctx context.Context, id int64) error {
	if err := s.userRepo.Delete(ctx, id); err != nil {
		return err
	}
	return s.identityRepo.DeleteByUserID(ctx, id)
}

// CreateFederatedUser 同一事务内建用户与外部身份行
// 身份已存在 → database.ErrIdentityExists；用户名被占 → database.ErrUsernameTaken
func (s *UserService) CreateFederatedUser(ctx context.Context, user *database.User, issuer, subject string) error {
	return s.identityRepo.CreateUserWithIdentity(ctx, user, issuer, subject)
}

func (s *UserService) UpdateUserLastLogin(ctx context.Context, id int64, ip string) error {
	return s.userRepo.UpdateLastLogin(ctx, id, ip)
}

// atlhyper_master_v2/service/operations/service_account_token.go
// OpsServiceAccounts 实现 — 服务账号的新建、修改、删除，访问令牌的签发、吊销与最近使用时间
package operations

import (
	"context"
	"fmt"
	"strings"
	"time"

	"AtlHyper/atlhyper_master_v2/auth/apitoken"
	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/common/logger"
)

var serviceAccountLog = logger.Module("ServiceAccount")

// ServiceAccountWriter 服务账号的写入部分（列表在 service/query）
type ServiceAccountWriter interface {
	Create(ctx context.Context, account *database.User) error
	GetByID(ctx context.Context, id int64) (*database.User, error)
	Update(ctx context.Context, account *database.User) error
	Delete(ctx context.Context, id int64) error
}

// APITokenWriter 访问令牌的写入部分
type APITokenWriter interface {
	CreateWithinLimit(ctx context.Context, token *database.APIToken, limit int) error
	Revoke(ctx context.Context, userID, tokenID int64, at time.Time) error
	Touch(ctx context.Context, id int64, usedAt time.Time) error
}

// ServiceAccountService 服务账号与访问令牌的写入操作
type ServiceAccountService struct {
	accounts ServiceAccountWriter
	tokens   APITokenWriter
	// 令牌生成与时钟可替换，便于测试
	generateToken func() (apitoken.Issued, error)
	now           func() time.Time
}

// NewServiceAccountService 创建 ServiceAccountService
func NewServiceAccountService(accounts ServiceAccountWriter, tokens APITokenWriter) *ServiceAccountService {
	return &ServiceAccountService{
		accounts:      accounts,
		tokens:        tokens,
		generateToken: apitoken.Generate,
		now:           time.Now,
	}
}

// CreateServiceAccount 新建服务账号（新建即启用），返回它的 id。
//
// 输入不合规 → model.ErrInvalidServiceAccountInput；账号名与任何用户名重复 → database.ErrUsernameTaken
func (s *ServiceAccountService) CreateServiceAccount(ctx context.Context, req *model.CreateServiceAccountRequest) (int64, error) {
	if err := model.ValidateServiceAccountName(req.Name); err != nil {
		return 0, err
	}
	displayName := strings.TrimSpace(req.DisplayName)
	if err := model.ValidateServiceAccountProfile(displayName, req.Role); err != nil {
		return 0, err
	}

	account := &database.User{
		Username:    req.Name,
		DisplayName: displayName,
		Role:        *req.Role,
		Status:      database.UserStatusActive,
	}
	if err := s.accounts.Create(ctx, account); err != nil {
		return 0, err
	}
	serviceAccountLog.Info("服务账号已新建", "account", account.ID, "role", account.Role)
	return account.ID, nil
}

// UpdateServiceAccount 修改显示名、等级与启停（整体替换）。改等级、停用对已签发的令牌立即生效。
//
// 输入不合规 → model.ErrInvalidServiceAccountInput；不存在 → database.ErrServiceAccountNotFound
func (s *ServiceAccountService) UpdateServiceAccount(ctx context.Context, id int64, req *model.UpdateServiceAccountRequest) error {
	trimmed := *req
	if req.DisplayName != nil {
		displayName := strings.TrimSpace(*req.DisplayName)
		trimmed.DisplayName = &displayName
	}
	if err := model.ValidateUpdateServiceAccountRequest(&trimmed); err != nil {
		return err
	}

	status := database.UserStatusDisabled
	if *trimmed.Enabled {
		status = database.UserStatusActive
	}
	account := &database.User{ID: id, DisplayName: *trimmed.DisplayName, Role: *trimmed.Role, Status: status}
	if err := s.accounts.Update(ctx, account); err != nil {
		return err
	}
	serviceAccountLog.Info("服务账号已修改", "account", id, "role", account.Role, "enabled", *req.Enabled)
	return nil
}

// DeleteServiceAccount 删除服务账号，它的令牌随之删除。不存在 → database.ErrServiceAccountNotFound
func (s *ServiceAccountService) DeleteServiceAccount(ctx context.Context, id int64) error {
	if err := s.accounts.Delete(ctx, id); err != nil {
		return err
	}
	serviceAccountLog.Info("服务账号已删除", "account", id)
	return nil
}

// IssueAPIToken 给服务账号签发一枚访问令牌；issuedBy 是签发人的用户名。
//
// 入库的只有哈希与前缀；明文只经返回值交给调用方这一次，之后无处可取。⛔ 明文与哈希都不写日志。
// 输入不合规 → model.ErrInvalidServiceAccountInput；账号不存在 → database.ErrServiceAccountNotFound；
// 未吊销的令牌已达上限 → database.ErrAPITokenLimitReached
func (s *ServiceAccountService) IssueAPIToken(ctx context.Context, accountID int64, req *model.IssueAPITokenRequest, issuedBy string) (*model.APITokenIssued, error) {
	name := strings.TrimSpace(req.Name)
	if err := model.ValidateAPITokenRequest(name, req.ExpiresInDays); err != nil {
		return nil, err
	}
	account, err := s.accounts.GetByID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	if account == nil {
		return nil, database.ErrServiceAccountNotFound
	}

	issued, err := s.generateToken()
	if err != nil {
		return nil, fmt.Errorf("生成访问令牌: %w", err)
	}
	// 库里的时间按秒存：这里先取整，返回给调用方的到期时间与库里的一致
	now := s.now().UTC().Truncate(time.Second)
	token := &database.APIToken{
		UserID:      accountID,
		Name:        name,
		TokenHash:   issued.Hash,
		TokenPrefix: issued.Prefix,
		CreatedBy:   issuedBy,
		CreatedAt:   now,
	}
	if req.ExpiresInDays != nil {
		expiresAt := now.AddDate(0, 0, *req.ExpiresInDays)
		token.ExpiresAt = &expiresAt
	}
	if err := s.tokens.CreateWithinLimit(ctx, token, model.MaxActiveAPITokensPerAccount); err != nil {
		return nil, err
	}
	serviceAccountLog.Info("已签发访问令牌", "account", accountID, "token", token.ID)

	result := &model.APITokenIssued{Token: issued.Plaintext, TokenPrefix: issued.Prefix}
	if token.ExpiresAt != nil {
		result.ExpiresAt = token.ExpiresAt.Format(time.RFC3339)
	}
	return result, nil
}

// RevokeAPIToken 吊销服务账号的一枚令牌（行保留，立即失效）。该账号下没有这枚令牌 → database.ErrAPITokenNotFound
func (s *ServiceAccountService) RevokeAPIToken(ctx context.Context, accountID, tokenID int64) error {
	if err := s.tokens.Revoke(ctx, accountID, tokenID, s.now()); err != nil {
		return err
	}
	serviceAccountLog.Info("已吊销访问令牌", "account", accountID, "token", tokenID)
	return nil
}

// TouchAPIToken 记下令牌的最近使用时间（访问令牌校验用；节流由校验方按库里的值判断）
func (s *ServiceAccountService) TouchAPIToken(ctx context.Context, id int64, usedAt time.Time) error {
	return s.tokens.Touch(ctx, id, usedAt)
}

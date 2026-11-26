package apitoken

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"AtlHyper/atlhyper_master_v2/auth/opaquetoken"
	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/common/logger"
)

var log = logger.Module("APIToken")

// lastUsedInterval 最近使用时间的记录粒度：距库里记下的上一次超过这么久才再写一次，
// 不让每个请求都写库
const lastUsedInterval = time.Minute

// ErrInvalidToken 令牌无效：空、格式不对、不存在、已吊销、已过期、账号不存在、不是服务账号、账号已停用、等级超出服务账号的上限。
// 返回的错误会包上原因（只供日志）；调用方用 errors.Is 判断，⛔ 不看文字
var ErrInvalidToken = errors.New("访问令牌无效")

// TokenStore 校验用到的最小存储接口
type TokenStore interface {
	GetAPITokenByHash(ctx context.Context, tokenHash string) (*database.APIToken, error) // 不存在返回 (nil, nil)
	TouchAPIToken(ctx context.Context, id int64, usedAt time.Time) error
}

// UserLoader 按 id 读账号（任何种类）
type UserLoader interface {
	GetUserByID(ctx context.Context, id int64) (*database.User, error) // 不存在返回 (nil, nil)
}

// Caller 令牌校验通过后的调用者
type Caller struct {
	UserID    int64
	Username  string
	Level     int // 账号当前的角色等级（每次校验时从库里读）
	TokenID   int64
	TokenName string
	ExpiresAt time.Time // 零值 = 不过期
}

// Authenticator 校验访问令牌
type Authenticator struct {
	store TokenStore
	users UserLoader
	now   func() time.Time
}

// NewAuthenticator 创建 Authenticator；now 为 nil 时用 time.Now
func NewAuthenticator(store TokenStore, users UserLoader, now func() time.Time) *Authenticator {
	if now == nil {
		now = time.Now
	}
	return &Authenticator{store: store, users: users, now: now}
}

// Authenticate 校验出示的令牌明文，返回调用者。
//
// 令牌无效 → 能用 errors.Is 认出 ErrInvalidToken 的错误；查库出错 → 原样返回那个错误
// （查失败与没查到是两回事：上层据此回 5xx 而不是 401）。
// ⛔ 令牌明文与哈希都不进错误信息与日志。
func (a *Authenticator) Authenticate(ctx context.Context, plaintext string) (Caller, error) {
	if !strings.HasPrefix(plaintext, plaintextPrefix) || len(plaintext) == len(plaintextPrefix) {
		return Caller{}, invalid("格式不对")
	}

	token, err := a.store.GetAPITokenByHash(ctx, Hash(plaintext))
	if err != nil {
		return Caller{}, fmt.Errorf("查询访问令牌: %w", err)
	}
	// 按哈希查到之后再常数时间比一次：存储不该返回别的行，这里不信任它
	if token == nil || !opaquetoken.Matches(plaintext, token.TokenHash) {
		return Caller{}, invalid("不存在")
	}

	now := a.now()
	if token.IsRevoked() {
		return Caller{}, invalid("已吊销")
	}
	if token.IsExpired(now) {
		return Caller{}, invalid("已过期")
	}

	account, err := a.users.GetUserByID(ctx, token.UserID)
	if err != nil {
		return Caller{}, fmt.Errorf("查询令牌所属账号: %w", err)
	}
	switch {
	case account == nil:
		return Caller{}, invalid("账号不存在")
	case !account.IsServiceAccount():
		return Caller{}, invalid("账号不是服务账号")
	case account.Status != database.UserStatusActive:
		return Caller{}, invalid("账号已停用")
	case account.Role < 0 || account.Role > model.ServiceAccountMaxRole:
		// 写入路径不会产生这样的行；库被直接改过时宁可拒绝，也不把更高的权限交给令牌
		return Caller{}, invalid("等级超出服务账号的上限")
	}

	a.recordUse(ctx, token, now)

	caller := Caller{
		UserID:    account.ID,
		Username:  account.Username,
		Level:     account.Role,
		TokenID:   token.ID,
		TokenName: token.Name,
	}
	if token.ExpiresAt != nil {
		caller.ExpiresAt = *token.ExpiresAt
	}
	return caller, nil
}

// recordUse 记最近使用时间：库里那一行距今超过 lastUsedInterval 才写。写失败只记告警，不影响校验结果
func (a *Authenticator) recordUse(ctx context.Context, token *database.APIToken, now time.Time) {
	if token.LastUsedAt != nil && now.Sub(*token.LastUsedAt) <= lastUsedInterval {
		return
	}
	if err := a.store.TouchAPIToken(ctx, token.ID, now); err != nil {
		log.Warn("记录访问令牌最近使用时间失败", "tokenId", token.ID, "err", err)
	}
}

func invalid(reason string) error {
	return fmt.Errorf("%w：%s", ErrInvalidToken, reason)
}

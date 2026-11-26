// atlhyper_master_v2/service/query/service_account_token.go
// QueryServiceAccounts 实现 — 服务账号管理页的列表（账号 + 令牌元数据 + 后端判定的状态），
// 以及访问令牌校验要用的按哈希取令牌
package query

import (
	"context"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/atlhyper_master_v2/model/convert"
)

// ServiceAccountReader 服务账号的只读部分（写入在 service/operations）
type ServiceAccountReader interface {
	List(ctx context.Context) ([]*database.User, error)
	GetByID(ctx context.Context, id int64) (*database.User, error)
}

// APITokenReader 访问令牌的只读部分（签发、吊销、记最近使用在 service/operations）
type APITokenReader interface {
	List(ctx context.Context) ([]*database.APIToken, error)
	ListByUser(ctx context.Context, userID int64) ([]*database.APIToken, error)
	GetByHash(ctx context.Context, tokenHash string) (*database.APIToken, error)
}

// ListServiceAccounts 全部服务账号（按账号名排序），各带令牌元数据；令牌状态与「能否签发」按当前时间判定
func (q *ServiceAccountQuery) ListServiceAccounts(ctx context.Context) ([]model.ServiceAccountView, error) {
	accounts, err := q.accounts.List(ctx)
	if err != nil {
		return nil, err
	}
	tokens, err := q.tokens.List(ctx)
	if err != nil {
		return nil, err
	}

	tokensByAccount := make(map[int64][]*database.APIToken, len(accounts))
	for _, token := range tokens {
		tokensByAccount[token.UserID] = append(tokensByAccount[token.UserID], token)
	}
	now := q.clock()
	views := make([]model.ServiceAccountView, 0, len(accounts))
	for _, account := range accounts {
		views = append(views, serviceAccountView(account, tokensByAccount[account.ID], now))
	}
	return views, nil
}

// GetServiceAccount 单个服务账号；不存在（或该 id 不是服务账号）时返回 (nil, nil)
func (q *ServiceAccountQuery) GetServiceAccount(ctx context.Context, id int64) (*model.ServiceAccountView, error) {
	account, err := q.accounts.GetByID(ctx, id)
	if err != nil || account == nil {
		return nil, err
	}
	tokens, err := q.tokens.ListByUser(ctx, id)
	if err != nil {
		return nil, err
	}
	view := serviceAccountView(account, tokens, q.clock())
	return &view, nil
}

// GetAPITokenByHash 按哈希取令牌（访问令牌校验用）；不存在返回 (nil, nil)
func (q *ServiceAccountQuery) GetAPITokenByHash(ctx context.Context, tokenHash string) (*database.APIToken, error) {
	return q.tokens.GetByHash(ctx, tokenHash)
}

// serviceAccountView 能否再签发 = 未吊销的令牌少于上限（与签发时仓库按同一条件判断：revoked_at 为空即计入）
func serviceAccountView(account *database.User, tokens []*database.APIToken, now time.Time) model.ServiceAccountView {
	unrevoked := 0
	for _, token := range tokens {
		if !token.IsRevoked() {
			unrevoked++
		}
	}
	return convert.ServiceAccountView(account, tokens, unrevoked < model.MaxActiveAPITokensPerAccount, now)
}

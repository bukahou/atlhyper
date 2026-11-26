// atlhyper_master_v2/model/convert/service_account_token.go
// database.User（服务账号）+ database.APIToken → model.ServiceAccountView 转换函数
package convert

import (
	"time"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/model"
)

// ServiceAccountView 把一个服务账号与它的令牌拼成管理页的一行。
//
// 令牌的状态按 now 判定；canIssueToken（未吊销的是否没到上限）由调用方按业务上限算好传入。
// 令牌哈希不写入结果。
func ServiceAccountView(account *database.User, tokens []*database.APIToken, canIssueToken bool, now time.Time) model.ServiceAccountView {
	view := model.ServiceAccountView{
		ID:            account.ID,
		Name:          account.Username,
		DisplayName:   account.DisplayName,
		Role:          account.Role,
		Enabled:       account.Status == database.UserStatusActive,
		CreatedAt:     formatUTC(account.CreatedAt),
		CanIssueToken: canIssueToken,
		Tokens:        make([]model.APITokenView, 0, len(tokens)),
	}
	var lastUsed time.Time
	for _, token := range tokens {
		view.Tokens = append(view.Tokens, APITokenView(token, now))
		if token.LastUsedAt != nil && token.LastUsedAt.After(lastUsed) {
			lastUsed = *token.LastUsedAt
		}
	}
	view.LastUsedAt = formatUTC(lastUsed)
	return view
}

// APITokenView 一枚令牌的元数据与按 now 判定的状态（吊销优先于过期）
func APITokenView(token *database.APIToken, now time.Time) model.APITokenView {
	status := model.APITokenStatusActive
	switch {
	case token.IsRevoked():
		status = model.APITokenStatusRevoked
	case token.IsExpired(now):
		status = model.APITokenStatusExpired
	}
	return model.APITokenView{
		ID:          token.ID,
		Name:        token.Name,
		TokenPrefix: token.TokenPrefix,
		Status:      status,
		CreatedBy:   token.CreatedBy,
		CreatedAt:   formatUTC(token.CreatedAt),
		ExpiresAt:   formatOptionalUTC(token.ExpiresAt),
		LastUsedAt:  formatOptionalUTC(token.LastUsedAt),
		RevokedAt:   formatOptionalUTC(token.RevokedAt),
	}
}

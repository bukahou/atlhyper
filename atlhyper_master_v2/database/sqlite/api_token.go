// atlhyper_master_v2/database/sqlite/api_token.go
// SQLite APITokenDialect 实现（访问令牌：只存哈希与显示用前缀）
package sqlite

import (
	"database/sql"
	"fmt"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
)

type apiTokenDialect struct{}

const apiTokenColumns = "id, user_id, name, token_hash, token_prefix, expires_at, last_used_at, created_by, created_at, revoked_at"

// InsertWithinLimit 计数与写入在同一条语句里：SQLite 的写语句串行执行，并发签发也不会超过上限
func (d *apiTokenDialect) InsertWithinLimit(token *database.APIToken, limit int) (string, []any) {
	return `INSERT INTO api_tokens (user_id, name, token_hash, token_prefix, expires_at, created_by, created_at)
	SELECT ?, ?, ?, ?, ?, ?, ?
	WHERE (SELECT COUNT(*) FROM api_tokens WHERE user_id = ? AND revoked_at IS NULL) < ?`,
		[]any{token.UserID, token.Name, token.TokenHash, token.TokenPrefix, nullableStamp(token.ExpiresAt),
			token.CreatedBy, utcStamp(token.CreatedAt), token.UserID, limit}
}

func (d *apiTokenDialect) SelectByHash(tokenHash string) (string, []any) {
	return "SELECT " + apiTokenColumns + " FROM api_tokens WHERE token_hash = ?", []any{tokenHash}
}

func (d *apiTokenDialect) SelectAll() (string, []any) {
	return "SELECT " + apiTokenColumns + " FROM api_tokens ORDER BY user_id, id DESC", nil
}

func (d *apiTokenDialect) SelectByUser(userID int64) (string, []any) {
	return "SELECT " + apiTokenColumns + " FROM api_tokens WHERE user_id = ? ORDER BY id DESC", []any{userID}
}

// Revoke 已吊销的保持第一次的时间；命中的行数为 0 说明该账号下没有这枚令牌
func (d *apiTokenDialect) Revoke(userID, tokenID int64, at time.Time) (string, []any) {
	return `UPDATE api_tokens SET revoked_at = COALESCE(revoked_at, ?) WHERE id = ? AND user_id = ?`,
		[]any{utcStamp(at), tokenID, userID}
}

func (d *apiTokenDialect) Touch(id int64, usedAt time.Time) (string, []any) {
	return `UPDATE api_tokens SET last_used_at = ? WHERE id = ?`, []any{utcStamp(usedAt), id}
}

func (d *apiTokenDialect) ScanRow(rows *sql.Rows) (*database.APIToken, error) {
	token := &database.APIToken{}
	var expiresAt, lastUsedAt, revokedAt sql.NullString
	var createdAt string
	err := rows.Scan(&token.ID, &token.UserID, &token.Name, &token.TokenHash, &token.TokenPrefix,
		&expiresAt, &lastUsedAt, &token.CreatedBy, &createdAt, &revokedAt)
	if err != nil {
		return nil, err
	}
	if token.ExpiresAt, err = parseValidityStamp(expiresAt, "expires_at"); err != nil {
		return nil, err
	}
	if token.RevokedAt, err = parseValidityStamp(revokedAt, "revoked_at"); err != nil {
		return nil, err
	}
	token.LastUsedAt = parseNullableStamp(lastUsedAt)
	token.CreatedAt = parseStamp(createdAt)
	return token, nil
}

// parseValidityStamp 到期与吊销时间决定令牌是否有效：存了却解析不了按读取失败处理，
// ⛔ 不当成「没有」（那会把一枚该失效的令牌当成永不过期、未吊销）
func parseValidityStamp(column sql.NullString, name string) (*time.Time, error) {
	if !column.Valid {
		return nil, nil
	}
	at, err := time.Parse(time.RFC3339, column.String)
	if err != nil {
		return nil, fmt.Errorf("api_tokens.%s 不是 RFC 3339 时间: %w", name, err)
	}
	return &at, nil
}

// IsUnknownAccount 令牌的 user_id 指向的账号不存在
func (d *apiTokenDialect) IsUnknownAccount(err error) bool {
	return violatesForeignKey(err)
}

var _ database.APITokenDialect = (*apiTokenDialect)(nil)

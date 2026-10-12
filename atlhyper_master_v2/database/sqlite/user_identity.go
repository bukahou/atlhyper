// atlhyper_master_v2/database/sqlite/user_identity.go
// SQLite UserIdentityDialect 实现
package sqlite

import (
	"strings"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
)

type userIdentityDialect struct{}

func (d *userIdentityDialect) Insert(userID int64, issuer, subject string) (string, []any) {
	return `INSERT INTO user_identities (user_id, issuer, subject, created_at) VALUES (?, ?, ?, ?)`,
		[]any{userID, issuer, subject, time.Now().Format(time.RFC3339)}
}

func (d *userIdentityDialect) SelectUserID(issuer, subject string) (string, []any) {
	return `SELECT user_id FROM user_identities WHERE issuer = ? AND subject = ?`, []any{issuer, subject}
}

func (d *userIdentityDialect) SelectFederatedUserIDs() (string, []any) {
	return `SELECT DISTINCT user_id FROM user_identities`, nil
}

func (d *userIdentityDialect) DeleteByUserID(userID int64) (string, []any) {
	return `DELETE FROM user_identities WHERE user_id = ?`, []any{userID}
}

// ClassifyInsertError SQLite 的唯一约束错误文案形如 "UNIQUE constraint failed: users.username"
func (d *userIdentityDialect) ClassifyInsertError(err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "UNIQUE constraint failed: user_identities."):
		return database.ErrIdentityExists
	case strings.Contains(msg, "UNIQUE constraint failed: users.username"):
		return database.ErrUsernameTaken
	}
	return err
}

var _ database.UserIdentityDialect = (*userIdentityDialect)(nil)

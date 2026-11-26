-- 服务账号与访问令牌。
-- 账号分两种：人（human）与服务账号（service），同在 users 表，用户名不能重复。已有的行都是人。
-- 服务账号凭访问令牌调用对外入口；令牌只存 SHA-256 与显示用前缀，明文只在签发时返回一次。
-- 时间列与其他表一样存 UTC 的 RFC 3339（秒精度，Z 结尾）。

-- +goose Up
ALTER TABLE users ADD COLUMN kind TEXT NOT NULL DEFAULT 'human' CHECK (kind IN ('human', 'service'));

-- 删除账号时它的令牌一起删除（连接开了 foreign_keys）；吊销只写 revoked_at，行保留
CREATE TABLE api_tokens (
	id           INTEGER PRIMARY KEY AUTOINCREMENT,
	user_id      INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	name         TEXT NOT NULL,
	token_hash   TEXT NOT NULL UNIQUE,
	token_prefix TEXT NOT NULL,
	expires_at   TEXT,
	last_used_at TEXT,
	created_by   TEXT NOT NULL DEFAULT '',
	created_at   TEXT NOT NULL,
	revoked_at   TEXT
);

CREATE INDEX idx_api_tokens_user ON api_tokens(user_id);

-- +goose Down
-- 回到没有服务账号的第 5 版：令牌与服务账号一起删除，人的账号保留。
DROP INDEX IF EXISTS idx_api_tokens_user;
DROP TABLE IF EXISTS api_tokens;
DELETE FROM users WHERE kind = 'service';
ALTER TABLE users DROP COLUMN kind;

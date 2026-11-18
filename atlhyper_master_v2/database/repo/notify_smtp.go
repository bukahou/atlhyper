// atlhyper_master_v2/database/repo/notify_smtp.go
// NotifySMTPRepository 实现（全局发件设置，单行）。
//
// 整份设置（含 SMTP 密码）序列化后加密再落库，读出时解密。
// ⛔ 本文件的日志与错误都不带配置内容。
package repo

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/common/logger"
)

var notifySMTPLog = logger.Module("NotifySMTPRepo")

type notifySMTPRepo struct {
	db      *sql.DB
	dialect database.NotifySMTPDialect
}

func newNotifySMTPRepo(db *sql.DB, dialect database.NotifySMTPDialect) *notifySMTPRepo {
	return &notifySMTPRepo{db: db, dialect: dialect}
}

// smtpRow 一行的原始内容：密文与更新时间
type smtpRow struct {
	encryptedConfig string
	updatedAt       time.Time
}

func (r *notifySMTPRepo) scanRow(rows *sql.Rows) (smtpRow, error) {
	encryptedConfig, updatedAt, err := r.dialect.ScanRow(rows)
	return smtpRow{encryptedConfig: encryptedConfig, updatedAt: updatedAt}, err
}

// Get 没有设置时返回 (nil, nil)。
// 解不开（密钥换过、数据损坏）时同样按没有设置处理：管理员在界面上重新填写即可，发信会因未配置而失败并留下记录
func (r *notifySMTPRepo) Get(ctx context.Context) (*database.NotifySMTP, error) {
	query, args := r.dialect.Select()
	row, found, err := queryNotifyRow(ctx, r.db, r.scanRow, query, args...)
	if err != nil || !found {
		return nil, err
	}
	if globalEncryptor == nil {
		return nil, database.ErrNotifyEncryptionUnavailable
	}
	plain, err := globalEncryptor.Decrypt(row.encryptedConfig)
	if err != nil {
		notifySMTPLog.Error("发件设置无法解密，按未配置处理")
		return nil, nil
	}
	settings := &database.NotifySMTP{}
	if err := json.Unmarshal([]byte(plain), settings); err != nil {
		notifySMTPLog.Error("发件设置格式损坏，按未配置处理")
		return nil, nil
	}
	settings.UpdatedAt = row.updatedAt
	return settings, nil
}

func (r *notifySMTPRepo) Save(ctx context.Context, settings *database.NotifySMTP) error {
	if globalEncryptor == nil {
		return database.ErrNotifyEncryptionUnavailable
	}
	plain, err := json.Marshal(settings)
	if err != nil {
		return fmt.Errorf("序列化发件设置失败")
	}
	encrypted, err := globalEncryptor.Encrypt(string(plain))
	if err != nil {
		return fmt.Errorf("加密发件设置失败")
	}
	now := time.Now()
	query, args := r.dialect.Upsert(encrypted, now)
	if _, err := r.db.ExecContext(ctx, query, args...); err != nil {
		return err
	}
	settings.UpdatedAt = now.UTC().Truncate(time.Second)
	return nil
}

var _ database.NotifySMTPRepository = (*notifySMTPRepo)(nil)

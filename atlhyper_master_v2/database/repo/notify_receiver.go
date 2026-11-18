// atlhyper_master_v2/database/repo/notify_receiver.go
// NotifyReceiverRepository 实现。
//
// 接收人的配置里有凭据（Webhook 地址、SMTP 密码）：整份配置序列化后加密再落库，读出时解密。
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

var notifyReceiverLog = logger.Module("NotifyReceiverRepo")

type notifyReceiverRepo struct {
	db      *sql.DB
	dialect database.NotifyReceiverDialect
}

func newNotifyReceiverRepo(db *sql.DB, dialect database.NotifyReceiverDialect) *notifyReceiverRepo {
	return &notifyReceiverRepo{db: db, dialect: dialect}
}

func (r *notifyReceiverRepo) Create(ctx context.Context, receiver *database.NotifyReceiver) error {
	encryptedConfig, err := encryptReceiverConfig(receiver)
	if err != nil {
		return err
	}
	now := time.Now()
	query, args := r.dialect.Insert(receiver, encryptedConfig, now)
	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		if r.dialect.IsDuplicateName(err) {
			return database.ErrNotifyReceiverNameTaken
		}
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	receiver.ID = id
	receiver.CreatedAt = now.UTC().Truncate(time.Second)
	receiver.UpdatedAt = receiver.CreatedAt
	return nil
}

func (r *notifyReceiverRepo) Update(ctx context.Context, receiver *database.NotifyReceiver) error {
	encryptedConfig, err := encryptReceiverConfig(receiver)
	if err != nil {
		return err
	}
	now := time.Now()
	query, args := r.dialect.Update(receiver, encryptedConfig, now)
	affected, err := execNotifyAffected(ctx, r.db, query, args...)
	if err != nil {
		if r.dialect.IsDuplicateName(err) {
			return database.ErrNotifyReceiverNameTaken
		}
		return err
	}
	if affected == 0 {
		return database.ErrNotifyReceiverNotFound
	}
	receiver.UpdatedAt = now.UTC().Truncate(time.Second)
	return nil
}

func (r *notifyReceiverRepo) Delete(ctx context.Context, id int64) error {
	query, args := r.dialect.Delete(id)
	affected, err := execNotifyAffected(ctx, r.db, query, args...)
	if err != nil {
		return err
	}
	if affected == 0 {
		return database.ErrNotifyReceiverNotFound
	}
	return nil
}

func (r *notifyReceiverRepo) GetByID(ctx context.Context, id int64) (*database.NotifyReceiver, error) {
	query, args := r.dialect.SelectByID(id)
	receiver, found, err := queryNotifyRow(ctx, r.db, r.scanDecrypted, query, args...)
	if err != nil || !found {
		return nil, err
	}
	return receiver, nil
}

func (r *notifyReceiverRepo) List(ctx context.Context) ([]*database.NotifyReceiver, error) {
	query, args := r.dialect.SelectAll()
	return queryNotifyRows(ctx, r.db, r.scanDecrypted, query, args...)
}

// scanDecrypted 扫描一行并解密配置。
//
// 解不开（密钥换过、数据损坏）时不报错：接收人照常列出，只是没有配置，
// 管理员可以在界面上重新填写凭据；发送时会因配置不完整而失败并留下记录
func (r *notifyReceiverRepo) scanDecrypted(rows *sql.Rows) (*database.NotifyReceiver, error) {
	receiver, encryptedConfig, err := r.dialect.ScanRow(rows)
	if err != nil {
		return nil, err
	}
	if err := decryptReceiverConfig(receiver, encryptedConfig); err != nil {
		notifyReceiverLog.Error("接收人配置无法解密，按未配置处理", "id", receiver.ID, "type", receiver.Type)
	}
	return receiver, nil
}

// encryptReceiverConfig 按类型取出配置，序列化后加密
func encryptReceiverConfig(receiver *database.NotifyReceiver) (string, error) {
	if globalEncryptor == nil {
		return "", database.ErrNotifyEncryptionUnavailable
	}
	var config any
	switch receiver.Type {
	case database.NotifyReceiverSlack:
		config = receiver.Slack
		if receiver.Slack == nil {
			config = &database.SlackConfig{}
		}
	case database.NotifyReceiverEmail:
		config = receiver.Email
		if receiver.Email == nil {
			config = &database.EmailConfig{}
		}
	default:
		return "", fmt.Errorf("不支持的接收人类型: %s", receiver.Type)
	}
	plain, err := json.Marshal(config)
	if err != nil {
		return "", fmt.Errorf("序列化接收人配置失败")
	}
	encrypted, err := globalEncryptor.Encrypt(string(plain))
	if err != nil {
		return "", fmt.Errorf("加密接收人配置失败")
	}
	return encrypted, nil
}

// decryptReceiverConfig 解密并按类型填回 Slack / Email。错误信息不带密文与明文
func decryptReceiverConfig(receiver *database.NotifyReceiver, encryptedConfig string) error {
	if globalEncryptor == nil {
		return database.ErrNotifyEncryptionUnavailable
	}
	plain, err := globalEncryptor.Decrypt(encryptedConfig)
	if err != nil {
		return fmt.Errorf("解密失败")
	}
	switch receiver.Type {
	case database.NotifyReceiverSlack:
		config := &database.SlackConfig{}
		if err := json.Unmarshal([]byte(plain), config); err != nil {
			return fmt.Errorf("配置格式损坏")
		}
		receiver.Slack = config
	case database.NotifyReceiverEmail:
		config := &database.EmailConfig{}
		if err := json.Unmarshal([]byte(plain), config); err != nil {
			return fmt.Errorf("配置格式损坏")
		}
		receiver.Email = config
	}
	return nil
}

var _ database.NotifyReceiverRepository = (*notifyReceiverRepo)(nil)

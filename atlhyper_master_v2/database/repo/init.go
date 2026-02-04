// atlhyper_master_v2/database/repo/init.go
// Repository 注入
package repo

import (
	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/common/crypto"
)

// 全局加密器（用于 API Key 加密）
var globalEncryptor *crypto.Encryptor

// SetEncryptionSecret 设置加密密钥
// 必须在 Init() 之前调用
func SetEncryptionSecret(secret string) error {
	enc, err := crypto.NewEncryptor(secret)
	if err != nil {
		return err
	}
	globalEncryptor = enc
	return nil
}

// Init 注入所有 Repository 到 DB 实例
// 在 database.New() 之后调用
func Init(db *database.DB, dialect database.Dialect) {
	db.Audit = newAuditRepo(db.Conn, dialect.Audit())
	db.User = newUserRepo(db.Conn, dialect.User())
	db.Event = newEventRepo(db.Conn, dialect.Event())
	db.Notify = newNotifyRepo(db.Conn, dialect.Notify())
	db.Cluster = newClusterRepo(db.Conn, dialect.Cluster())
	db.Command = newCommandRepo(db.Conn, dialect.Command())
	db.Settings = newSettingsRepo(db.Conn, dialect.Settings())
	db.AIConversation = newAIConversationRepo(db.Conn, dialect.AIConversation())
	db.AIMessage = newAIMessageRepo(db.Conn, dialect.AIMessage())
	db.AIProvider = newAIProviderRepo(db.Conn, dialect.AIProvider())
	db.AIActive = newAIActiveConfigRepo(db.Conn, dialect.AIActiveConfig())
	db.AIModel = newAIProviderModelRepo(db.Conn, dialect.AIProviderModel())
	db.SLO = newSLORepo(db.Conn, dialect.SLO())
	db.NodeMetrics = newNodeMetricsRepo(db.Conn, dialect.NodeMetrics())
}

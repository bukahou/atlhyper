// atlhyper_master_v2/database/repo/init.go
// Repository 注入
package repo

import (
	"AtlHyper/atlhyper_master_v2/database"
)

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
}

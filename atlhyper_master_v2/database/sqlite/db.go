// atlhyper_master_v2/database/sqlite/db.go
// SQLite 数据库实现
package sqlite

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/database/repository"
	"AtlHyper/atlhyper_master_v2/database/sqlite/impl"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteDB SQLite 数据库
type SQLiteDB struct {
	db *sql.DB

	// Repository 实例
	clusterEventRepo   repository.ClusterEventRepository
	notifyChannelRepo  repository.NotifyChannelRepository
	userRepo           repository.UserRepository
	clusterRepo        repository.ClusterRepository
	auditRepo          repository.AuditRepository
	commandHistoryRepo repository.CommandHistoryRepository
	settingsRepo       repository.SettingsRepository
}

// New 创建 SQLite 数据库
func New(path string) (*SQLiteDB, error) {
	// 确保目录存在
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	// 打开数据库
	db, err := sql.Open("sqlite3", path+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, err
	}

	// 测试连接
	if err := db.Ping(); err != nil {
		return nil, err
	}

	sqliteDB := &SQLiteDB{db: db}

	// 初始化 Repository
	sqliteDB.clusterEventRepo = impl.NewClusterEventRepository(db)
	sqliteDB.notifyChannelRepo = impl.NewNotifyChannelRepository(db)
	sqliteDB.userRepo = impl.NewUserRepository(db)
	sqliteDB.clusterRepo = impl.NewClusterRepository(db)
	sqliteDB.auditRepo = impl.NewAuditRepository(db)
	sqliteDB.commandHistoryRepo = impl.NewCommandHistoryRepository(db)
	sqliteDB.settingsRepo = impl.NewSettingsRepository(db)

	log.Printf("[SQLiteDB] 已连接数据库: %s", path)
	return sqliteDB, nil
}

// Migrate 执行数据库迁移
func (s *SQLiteDB) Migrate() error {
	return migrate(s.db)
}

// Close 关闭数据库连接
func (s *SQLiteDB) Close() error {
	return s.db.Close()
}

// ==================== Repository 获取 ====================

func (s *SQLiteDB) ClusterEventRepository() repository.ClusterEventRepository {
	return s.clusterEventRepo
}

func (s *SQLiteDB) NotifyChannelRepository() repository.NotifyChannelRepository {
	return s.notifyChannelRepo
}

func (s *SQLiteDB) UserRepository() repository.UserRepository {
	return s.userRepo
}

func (s *SQLiteDB) ClusterRepository() repository.ClusterRepository {
	return s.clusterRepo
}

func (s *SQLiteDB) AuditRepository() repository.AuditRepository {
	return s.auditRepo
}

func (s *SQLiteDB) CommandHistoryRepository() repository.CommandHistoryRepository {
	return s.commandHistoryRepo
}

func (s *SQLiteDB) SettingsRepository() repository.SettingsRepository {
	return s.settingsRepo
}

// 确保实现了接口
var _ database.Database = (*SQLiteDB)(nil)

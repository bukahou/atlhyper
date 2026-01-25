// atlhyper_master/repository/registry.go
// 仓库层全局实例注册
package repository

import "errors"

// ============================================================
// SQL 仓库全局实例
// ============================================================

var (
	User    UserRepository
	Audit   AuditRepository
	Event   EventRepository
	Config  ConfigRepository
	Metrics MetricsRepository
)

// ============================================================
// 内存仓库全局实例
// ============================================================

var (
	Mem    MemReader // 内存读取
	MemW   MemWriter // 内存写入
)

// ============================================================
// 初始化函数
// ============================================================

// InitSQL 初始化 SQL 仓库实例
func InitSQL(
	user UserRepository,
	audit AuditRepository,
	event EventRepository,
	config ConfigRepository,
	metrics MetricsRepository,
) {
	User = user
	Audit = audit
	Event = event
	Config = config
	Metrics = metrics
}

// InitMem 初始化内存仓库实例
func InitMem(reader MemReader, writer MemWriter) {
	Mem = reader
	MemW = writer
}

// ============================================================
// 验证函数
// ============================================================

// ValidateSQL 检查 SQL 仓库是否已初始化
func ValidateSQL() error {
	if User == nil {
		return errors.New("UserRepository not initialized")
	}
	if Audit == nil {
		return errors.New("AuditRepository not initialized")
	}
	if Event == nil {
		return errors.New("EventRepository not initialized")
	}
	if Config == nil {
		return errors.New("ConfigRepository not initialized")
	}
	if Metrics == nil {
		return errors.New("MetricsRepository not initialized")
	}
	return nil
}

// ValidateMem 检查内存仓库是否已初始化
func ValidateMem() error {
	if Mem == nil {
		return errors.New("MemReader not initialized")
	}
	if MemW == nil {
		return errors.New("MemWriter not initialized")
	}
	return nil
}

// Validate 检查所有仓库是否已初始化
func Validate() error {
	if err := ValidateSQL(); err != nil {
		return err
	}
	return ValidateMem()
}

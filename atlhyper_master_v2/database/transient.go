package database

import (
	"context"
	"errors"
	"strings"
)

// IsTransientWriteError 判断一次写入失败是否是「暂时性」的：过一会儿重试有望成功。
//
// 暂时性失败指写锁被占住等不到（SQLITE_BUSY / LOCKED），或等锁期间超时。
// 典型场景是存储卡顿：正在落盘的那次写入占着锁，其余写入全部等到超时。
// 约束冲突、语法错误等不属于此类，重试也不会成功。
//
// SQLite 的识别靠错误文本（驱动不导出稳定的错误类型给上层）；
// transient_test.go 用真实的锁冲突守护这段文本匹配。
func IsTransientWriteError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	message := err.Error()
	return strings.Contains(message, "SQLITE_BUSY") ||
		strings.Contains(message, "SQLITE_LOCKED") ||
		strings.Contains(message, "database is locked") ||
		strings.Contains(message, "database table is locked")
}

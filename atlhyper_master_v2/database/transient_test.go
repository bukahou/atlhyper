package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"testing"
	"time"
)

// 用真实的 SQLite 造一次锁冲突，确认驱动返回的错误确实被识别为「暂时性」。
// 识别靠的是错误文本，驱动换版本后文本若变了，这里会失败。
func TestIsTransientWriteError_RealBusyError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "busy.db")
	holder, err := sql.Open("sqlite", path+"?_pragma=journal_mode(WAL)")
	if err != nil {
		t.Fatal(err)
	}
	defer holder.Close()
	if _, err := holder.Exec(`CREATE TABLE t (id INTEGER PRIMARY KEY, v TEXT)`); err != nil {
		t.Fatal(err)
	}

	// 一个连接占住写锁不放
	ctx := context.Background()
	conn, err := holder.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, `BEGIN IMMEDIATE`); err != nil {
		t.Fatal(err)
	}
	defer conn.ExecContext(ctx, `ROLLBACK`)

	// 另一个连接只等 50ms 就放弃
	waiter, err := sql.Open("sqlite", path+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(50)")
	if err != nil {
		t.Fatal(err)
	}
	defer waiter.Close()

	_, busyErr := waiter.Exec(`INSERT INTO t (v) VALUES ('x')`)
	if busyErr == nil {
		t.Fatal("期望写入因锁冲突失败")
	}
	if !IsTransientWriteError(busyErr) {
		t.Errorf("锁冲突应识别为暂时性错误，实际错误文本: %v", busyErr)
	}

	// 等锁的过程中 ctx 超时，同样是暂时性的
	deadlineCtx, cancel := context.WithTimeout(ctx, 20*time.Millisecond)
	defer cancel()
	// 驱动在等锁期间不会立刻响应 ctx 取消，要等到 busy_timeout 到点才返回；这里设短一些让测试快
	slowWaiter, err := sql.Open("sqlite", path+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(300)")
	if err != nil {
		t.Fatal(err)
	}
	defer slowWaiter.Close()
	_, deadlineErr := slowWaiter.ExecContext(deadlineCtx, `INSERT INTO t (v) VALUES ('y')`)
	if deadlineErr == nil {
		t.Fatal("期望写入因超时失败")
	}
	if !IsTransientWriteError(deadlineErr) {
		t.Errorf("等锁超时应识别为暂时性错误，实际错误: %v", deadlineErr)
	}
}

func TestIsTransientWriteError_OtherErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"包装过的超时", fmt.Errorf("update state: %w", context.DeadlineExceeded), true},
		{"约束冲突", errors.New("constraint failed: UNIQUE constraint failed: aiops_incidents.id (1555)"), false},
		{"语法错误", errors.New("SQL logic error: near \"FORM\": syntax error (1)"), false},
		{"调用方取消", context.Canceled, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsTransientWriteError(tt.err); got != tt.want {
				t.Errorf("IsTransientWriteError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

// atlhyper_master_v2/aiops/incident/write_queue.go
// 事件写入的重试队列
package incident

import (
	"context"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
)

const (
	// maxPendingWrites 待重试写入的上限；超出时丢最旧的，存储长时间不可用也不会无限占内存
	maxPendingWrites = 1000
	// retryBackoff 一次暂时性失败之后，隔多久再碰数据库
	retryBackoff = 10 * time.Second
)

// pendingWrite 一步数据库写入。每步只做一件事，失败了可以单独重试。
type pendingWrite struct {
	describe string
	run      func(ctx context.Context) error
}

// submit 把若干步写入按顺序放进队列并尝试执行。
//
// 事件的状态只在内存里的状态机往前走；如果写库失败就丢弃，库里和页面上会一直停在旧状态。
// 所以写入一律先入队：
//   - 平时队列是空的，入队后立刻执行完，与直接写没有区别
//   - 写锁被占住（存储卡顿）时，失败的那步留在队首，其后的写入排在它后面，保持先后顺序
//   - 失败后的 retryBackoff 内不再尝试：每次尝试都要等满写锁超时，逐条去试会拖住整条快照处理
//   - 重试也不会成功的错误（约束冲突等）记日志后丢弃，不堵住后面的写入
func (s *Store) submit(ctx context.Context, writes ...pendingWrite) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	s.pending = append(s.pending, writes...)
	if overflow := len(s.pending) - maxPendingWrites; overflow > 0 {
		log.Error("待重试的事件写入超过上限，丢弃最旧的", "dropped", overflow, "limit", maxPendingWrites)
		s.pending = append([]pendingWrite(nil), s.pending[overflow:]...)
	}
	s.flushLocked(ctx)
}

// FlushPending 重试队列里积压的写入。由引擎在每份快照到来时调用；队列为空时没有开销。
func (s *Store) FlushPending(ctx context.Context) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	s.flushLocked(ctx)
}

// PendingWrites 当前积压的写入步数
func (s *Store) PendingWrites() int {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	return len(s.pending)
}

func (s *Store) flushLocked(ctx context.Context) {
	if s.now().Before(s.retryAfter) {
		return
	}
	hadBacklog := !s.retryAfter.IsZero()

	for len(s.pending) > 0 {
		write := s.pending[0]
		err := write.run(ctx)
		if err != nil && database.IsTransientWriteError(err) {
			s.retryAfter = s.now().Add(retryBackoff)
			log.Warn("事件写入暂时失败，稍后重试", "write", write.describe, "pending", len(s.pending), "err", err)
			return
		}
		if err != nil {
			log.Error("事件写入失败，已放弃", "write", write.describe, "err", err)
		}
		s.pending[0] = pendingWrite{}
		s.pending = s.pending[1:]
	}

	if hadBacklog {
		log.Info("积压的事件写入已全部补上")
	}
	s.retryAfter = time.Time{}
	s.pending = nil
}

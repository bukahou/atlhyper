package memory

import (
	"context"
	"errors"
	"testing"
	"time"

	mq "AtlHyper/atlhyper_master_v2/mq/contract"
	"AtlHyper/model_v3/command"
)

func enqueued(t *testing.T, bus *MemoryBus, clusterID, cmdID string) {
	t.Helper()
	if err := bus.EnqueueCommand(clusterID, "ops", &command.Command{ID: cmdID, ClusterID: clusterID}); err != nil {
		t.Fatal(err)
	}
}

// 只有指令所属的集群能回报它的结果；别的集群回报时不写入、原指令保持未完成。
func TestAckCommand_OnlyOwningClusterMayAck(t *testing.T) {
	bus := NewMemoryBus()
	enqueued(t, bus, "c1", "cmd-1")

	err := bus.AckCommand("c2", "cmd-1", &command.Result{CommandID: "cmd-1", Success: true, Output: "forged"})
	if !errors.Is(err, command.ErrNotOwnedByCluster) {
		t.Fatalf("别的集群回报应返回 ErrNotOwnedByCluster，实际 %v", err)
	}
	status, _ := bus.GetCommandStatus("cmd-1")
	if status == nil || status.Result != nil || status.Status != command.StatusPending {
		t.Fatalf("被拒绝的回报不应改变指令状态，实际 %+v", status)
	}

	if err := bus.AckCommand("c1", "cmd-1", &command.Result{CommandID: "cmd-1", Success: true}); err != nil {
		t.Fatalf("所属集群回报失败: %v", err)
	}
	status, _ = bus.GetCommandStatus("cmd-1")
	if status == nil || status.Status != command.StatusSuccess {
		t.Fatalf("所属集群回报后应为成功，实际 %+v", status)
	}
}

// 集群参数为空表示不核对归属：Master 自己判定指令过期时这样调用。
func TestAckCommand_EmptyClusterSkipsOwnershipCheck(t *testing.T) {
	bus := NewMemoryBus()
	enqueued(t, bus, "c1", "cmd-1")

	if err := bus.AckCommand("", "cmd-1", &command.Result{CommandID: "cmd-1", Success: false, Error: "expired"}); err != nil {
		t.Fatalf("不核对归属时应成功，实际 %v", err)
	}
}

// 话题只有 ops / ai 两个：别的话题不建队列
func TestEnqueueCommand_RejectsUnknownTopic(t *testing.T) {
	bus := NewMemoryBus()
	err := bus.EnqueueCommand("c1", "anything", &command.Command{ID: "x", ClusterID: "c1"})
	if !errors.Is(err, mq.ErrUnknownTopic) {
		t.Fatalf("err = %v, want ErrUnknownTopic", err)
	}
	if _, err := bus.WaitCommand(context.Background(), "c1", "anything", time.Millisecond); !errors.Is(err, mq.ErrUnknownTopic) {
		t.Fatalf("WaitCommand err = %v, want ErrUnknownTopic", err)
	}
	if len(bus.queues) != 0 {
		t.Errorf("未知话题不应建队列，实际 %d 个", len(bus.queues))
	}
}

// 单个队列有长度上限：没人取的指令不会无限堆积
func TestEnqueueCommand_QueueFull(t *testing.T) {
	bus := NewMemoryBus()
	bus.maxQueueLen = 2
	enqueued(t, bus, "c1", "a")
	enqueued(t, bus, "c1", "b")
	err := bus.EnqueueCommand("c1", mq.TopicOps, &command.Command{ID: "c", ClusterID: "c1"})
	if !errors.Is(err, mq.ErrQueueFull) {
		t.Fatalf("err = %v, want ErrQueueFull", err)
	}
	if status, _ := bus.GetCommandStatus("c"); status != nil {
		t.Error("没入队的指令不应留下状态记录")
	}
}

// 清理：空闲的空队列删掉；一直没完成的状态记录过了上限也删掉（取不到结果的指令不会永远占着内存）
func TestCleanup_IdleQueuesAndStaleStatuses(t *testing.T) {
	bus := NewMemoryBus()
	now := time.Now()

	// 空队列：被长轮询建出来后没人再用
	if _, err := bus.WaitCommand(context.Background(), "ghost", mq.TopicOps, time.Millisecond); err != nil {
		t.Fatal(err)
	}
	// 有指令、但一直没被取走
	enqueued(t, bus, "c1", "stuck")

	bus.cleanup(now.Add(bus.idleQueueTTL + time.Minute))
	if _, ok := bus.queues[queueKey("ghost", mq.TopicOps)]; ok {
		t.Error("空闲的空队列应被删除")
	}
	if _, ok := bus.queues[queueKey("c1", mq.TopicOps)]; !ok {
		t.Error("还有指令的队列不应被删除")
	}

	bus.cleanup(now.Add(bus.staleStatusTTL + time.Minute))
	if status, _ := bus.GetCommandStatus("stuck"); status != nil {
		t.Error("超过上限仍未完成的状态记录应被删除")
	}
}

// 等不到结果是「超时」，不是「没有结果」：返回明确的错误，不返回 (nil, nil)
func TestWaitCommandResult_TimeoutIsAnError(t *testing.T) {
	bus := NewMemoryBus()
	enqueued(t, bus, "c1", "cmd-1")
	result, err := bus.WaitCommandResult(context.Background(), "cmd-1", 10*time.Millisecond)
	if result != nil {
		t.Fatalf("超时不应有结果，实际 %+v", result)
	}
	if !errors.Is(err, mq.ErrWaitTimeout) || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want ErrWaitTimeout（同时是 DeadlineExceeded）", err)
	}
}

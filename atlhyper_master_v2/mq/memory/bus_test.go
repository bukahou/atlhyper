package memory

import (
	"errors"
	"testing"

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

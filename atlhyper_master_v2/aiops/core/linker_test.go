package core

import (
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/aiops"
	"AtlHyper/atlhyper_master_v2/aiops/grouping"
)

// 新归入、换源头、解除各产生一条变化；源头不变（只是路径不同）不算变化；各集群互不影响
func TestLinkState_Diff(t *testing.T) {
	var state linkState
	now := time.Date(2026, 11, 15, 7, 0, 0, 0, time.UTC)
	path := func(edge string) []aiops.LinkPathStep {
		return []aiops.LinkPathStep{{EntityKey: "a"}, {EntityKey: "b", Edge: edge}}
	}

	changes := state.diff("c1", map[string]grouping.Link{"inc-a": {RootIncidentID: "inc-r", Path: path("selects")}}, now)
	if len(changes) != 1 || changes[0].IncidentID != "inc-a" || changes[0].RootIncidentID != "inc-r" || changes[0].ClusterID != "c1" {
		t.Fatalf("新归入应产生一条变化: %+v", changes)
	}
	if changes := state.diff("c1", map[string]grouping.Link{"inc-a": {RootIncidentID: "inc-r", Path: path("calls")}}, now); len(changes) != 0 {
		t.Errorf("源头不变不算变化: %+v", changes)
	}
	if changes := state.diff("c2", map[string]grouping.Link{}, now); len(changes) != 0 {
		t.Errorf("别的集群不受影响: %+v", changes)
	}
	changes = state.diff("c1", map[string]grouping.Link{"inc-a": {RootIncidentID: "inc-r2"}}, now)
	if len(changes) != 1 || changes[0].RootIncidentID != "inc-r2" {
		t.Errorf("换源头应产生一条变化: %+v", changes)
	}
	changes = state.diff("c1", map[string]grouping.Link{}, now)
	if len(changes) != 1 || changes[0].IncidentID != "inc-a" || changes[0].RootIncidentID != "" {
		t.Errorf("解除应产生一条源头为空的变化: %+v", changes)
	}
}

// 启动恢复的关联在下一轮里照常比较：仍成立不重复发，不成立则解除
func TestLinkState_RestoreThenDiff(t *testing.T) {
	var state linkState
	now := time.Date(2026, 11, 15, 7, 0, 0, 0, time.UTC)
	state.restore([]aiops.IncidentLink{
		{ClusterID: "c1", IncidentID: "inc-a", RootIncidentID: "inc-r"},
		{ClusterID: "c1", IncidentID: "inc-b", RootIncidentID: "inc-r"},
	})
	changes := state.diff("c1", map[string]grouping.Link{"inc-a": {RootIncidentID: "inc-r"}}, now)
	if len(changes) != 1 || changes[0].IncidentID != "inc-b" || changes[0].RootIncidentID != "" {
		t.Errorf("恢复的关联仍成立不应重复发，不成立应解除: %+v", changes)
	}
}

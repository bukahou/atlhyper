package tools

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/ai/llm"
	"AtlHyper/atlhyper_master_v2/ai/toolkit"
	"AtlHyper/model_v3/agent"
	"AtlHyper/model_v3/cluster"
)

type fakeClusterLister struct {
	clusters []agent.ClusterInfo
	err      error
}

func (f fakeClusterLister) ListClusters(context.Context) ([]agent.ClusterInfo, error) {
	return f.clusters, f.err
}

type fakeOverviewReader struct {
	overviews map[string]*cluster.ClusterOverview
	err       error
}

func (f fakeOverviewReader) GetOverview(_ context.Context, clusterID string) (*cluster.ClusterOverview, error) {
	return f.overviews[clusterID], f.err
}

func overviewRegistry(d Deps) *toolkit.Registry {
	reg := toolkit.NewRegistry()
	reg.Register(overviewTools(d)...)
	return reg
}

func TestListClusters(t *testing.T) {
	seen := time.Date(2026, 11, 26, 4, 0, 0, 0, time.UTC)
	reg := overviewRegistry(Deps{Clusters: fakeClusterLister{clusters: []agent.ClusterInfo{
		{ClusterID: "c1", Status: "online", LastSeen: seen, NodeCount: 4, PodCount: 170, OTelAvailable: true},
	}}})
	tool, ok := reg.Lookup("list_clusters")
	if !ok {
		t.Fatal("list_clusters 没注册")
	}
	if tool.NeedsCluster() {
		t.Error("list_clusters 不针对某个集群")
	}
	if tool.Effect() != toolkit.EffectRead {
		t.Errorf("Effect = %q", tool.Effect())
	}

	// 不给集群也能调：处理函数不看 clusterID
	out, err := reg.Execute(context.Background(), "", allowAll, &llm.ToolCall{Name: "list_clusters", Params: "{}"})
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		Clusters []agent.ClusterInfo `json:"clusters"`
		Total    int                 `json:"total"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("结果不是 JSON: %v（%s）", err, out)
	}
	if got.Total != 1 || len(got.Clusters) != 1 || got.Clusters[0].ClusterID != "c1" || !got.Clusters[0].LastSeen.Equal(seen) {
		t.Errorf("结果 = %+v", got)
	}

	// 没有集群时给空列表，不是 null
	empty := overviewRegistry(Deps{Clusters: fakeClusterLister{}})
	out, err = empty.Execute(context.Background(), "", allowAll, &llm.ToolCall{Name: "list_clusters", Params: "{}"})
	if err != nil || out != `{"clusters":[],"total":0}` {
		t.Errorf("空结果 = (%s, %v)", out, err)
	}

	// 查失败是错误，不是空列表
	failing := overviewRegistry(Deps{Clusters: fakeClusterLister{err: errors.New("store down")}})
	if _, err := failing.Execute(context.Background(), "", allowAll, &llm.ToolCall{Name: "list_clusters", Params: "{}"}); err == nil {
		t.Error("列出集群失败应返回错误")
	}
}

func TestGetClusterOverview(t *testing.T) {
	overview := &cluster.ClusterOverview{ClusterID: "c1"}
	reg := overviewRegistry(Deps{Overview: fakeOverviewReader{overviews: map[string]*cluster.ClusterOverview{"c1": overview}}})

	out, err := reg.Execute(context.Background(), "c1", allowAll, &llm.ToolCall{Name: "get_cluster_overview", Params: "{}"})
	if err != nil {
		t.Fatal(err)
	}
	var got cluster.ClusterOverview
	if err := json.Unmarshal([]byte(out), &got); err != nil || got.ClusterID != "c1" {
		t.Errorf("结果 = (%s, %v)", out, err)
	}

	// 还没有数据的集群：明确报错，不返回空对象装成功
	if _, err := reg.Execute(context.Background(), "c2", allowAll, &llm.ToolCall{Name: "get_cluster_overview", Params: "{}"}); !errors.Is(err, ErrNoClusterData) {
		t.Errorf("没有数据应返回 ErrNoClusterData，实际 %v", err)
	}

	failing := overviewRegistry(Deps{Overview: fakeOverviewReader{err: errors.New("store down")}})
	if _, err := failing.Execute(context.Background(), "c1", allowAll, &llm.ToolCall{Name: "get_cluster_overview", Params: "{}"}); err == nil || errors.Is(err, ErrNoClusterData) {
		t.Errorf("查失败应是另一种错误，实际 %v", err)
	}
}

// 两个概览工具只要 0 级的 cluster.read，任何已识别的调用者都看得到
func TestOverviewTools_Permission(t *testing.T) {
	reg := overviewRegistry(Deps{})
	onlyClusterRead := func(p string) bool { return p == toolkit.PermClusterRead }
	defs := reg.Definitions(onlyClusterRead)
	if len(defs) != 2 {
		t.Fatalf("有 cluster.read 应看到 2 个概览工具，实际 %d", len(defs))
	}
	if got := reg.Definitions(func(string) bool { return false }); len(got) != 0 {
		t.Errorf("没有 cluster.read 不应看到概览工具，实际 %d", len(got))
	}
}

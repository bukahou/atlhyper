package grouping

import (
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/aiops"
)

var t0 = time.Date(2026, 11, 15, 7, 0, 0, 0, time.UTC)

// testGraph 依赖图：边 from → to 表示 from 依赖 to
func testGraph(edges ...[3]string) *aiops.DependencyGraph {
	g := aiops.NewDependencyGraph("c1")
	for _, e := range edges {
		g.AddEdge(e[0], e[1], e[2], 1.0)
	}
	g.RebuildIndex()
	return g
}

func cand(id, subject string, startOffset time.Duration, risk float64, members ...string) Candidate {
	return Candidate{IncidentID: id, SubjectKey: subject, StartedAt: t0.Add(startOffset), Risk: risk, Members: members}
}

// Deployment 崩溃、它的 Service 分到传播风险：Service 归到 Deployment 下，路径 Service → 选中 → Pod
func TestGroup_ServiceUnderItsWorkload(t *testing.T) {
	g := testGraph([3]string{"shop/service/api", "shop/pod/api-1", "selects"}, [3]string{"shop/service/api", "shop/pod/api-2", "selects"})
	links := Group([]Candidate{
		cand("inc-deploy", "shop/deployment/api", 0, 0.9, "shop/pod/api-1", "shop/pod/api-2"),
		cand("inc-svc", "shop/service/api", 30*time.Second, 0.36, "shop/service/api"),
	}, g)

	link, ok := links["inc-svc"]
	if !ok || link.RootIncidentID != "inc-deploy" {
		t.Fatalf("Service 应归到 Deployment 下: %+v", links)
	}
	if len(link.Path) != 2 || link.Path[0] != (aiops.LinkPathStep{EntityKey: "shop/service/api"}) ||
		link.Path[1].Edge != "selects" || link.Path[1].EntityKey != "shop/pod/api-1" {
		t.Errorf("路径不对: %+v", link.Path)
	}
	if _, linked := links["inc-deploy"]; linked {
		t.Error("源头自己不应有关联")
	}
}

// 链式：调用方 Service → 调用 → 数据库 Service → 选中 → 数据库 Pod：两者都挂到最上游的数据库 StatefulSet，不嵌套
func TestGroup_ChainAllUnderTopmost(t *testing.T) {
	g := testGraph(
		[3]string{"shop/service/api", "shop/service/db", "calls"},
		[3]string{"shop/service/db", "shop/pod/db-0", "selects"},
	)
	links := Group([]Candidate{
		cand("inc-db", "shop/statefulset/db", 0, 0.95, "shop/pod/db-0"),
		cand("inc-dbsvc", "shop/service/db", time.Minute, 0.4, "shop/service/db"),
		cand("inc-api", "shop/service/api", 2*time.Minute, 0.6, "shop/service/api"),
	}, g)
	if links["inc-dbsvc"].RootIncidentID != "inc-db" || links["inc-api"].RootIncidentID != "inc-db" {
		t.Fatalf("链上的都应直接挂到最上游: %+v", links)
	}
	if path := links["inc-api"].Path; len(path) != 3 || path[1].Edge != "calls" || path[2].Edge != "selects" ||
		path[2].EntityKey != "shop/pod/db-0" {
		t.Errorf("调用方到源头的路径不对: %+v", path)
	}
}

// 上游比下游晚 2 分钟以上出问题：不归（多半是各出各的）；2 分钟以内仍归（采集先后误差）
func TestGroup_UpstreamMustNotStartMuchLater(t *testing.T) {
	g := testGraph([3]string{"shop/service/api", "shop/pod/api-1", "selects"})
	late := Group([]Candidate{
		cand("inc-deploy", "shop/deployment/api", 5*time.Minute, 0.9, "shop/pod/api-1"),
		cand("inc-svc", "shop/service/api", 0, 0.36, "shop/service/api"),
	}, g)
	if len(late) != 0 {
		t.Errorf("上游晚 5 分钟不应归: %+v", late)
	}
	slightly := Group([]Candidate{
		cand("inc-deploy", "shop/deployment/api", 90*time.Second, 0.9, "shop/pod/api-1"),
		cand("inc-svc", "shop/service/api", 0, 0.36, "shop/service/api"),
	}, g)
	if slightly["inc-svc"].RootIncidentID != "inc-deploy" {
		t.Errorf("上游晚 90 秒仍应归: %+v", slightly)
	}
}

// 依赖走不到（超过 4 步）不归
func TestGroup_DepthLimit(t *testing.T) {
	g := testGraph(
		[3]string{"a/service/s1", "a/service/s2", "calls"},
		[3]string{"a/service/s2", "a/service/s3", "calls"},
		[3]string{"a/service/s3", "a/service/s4", "calls"},
		[3]string{"a/service/s4", "a/service/s5", "calls"},
		[3]string{"a/service/s5", "a/pod/p", "selects"},
	)
	far := Group([]Candidate{
		cand("inc-root", "a/deployment/p", 0, 0.9, "a/pod/p"),
		cand("inc-far", "a/service/s1", time.Minute, 0.5, "a/service/s1"),
	}, g)
	if _, linked := far["inc-far"]; linked {
		t.Errorf("5 步才到，不应归: %+v", far["inc-far"])
	}
	near := Group([]Candidate{
		cand("inc-root", "a/deployment/p", 0, 0.9, "a/pod/p"),
		cand("inc-near", "a/service/s3", time.Minute, 0.5, "a/service/s3"),
	}, g)
	if near["inc-near"].RootIncidentID != "inc-root" {
		t.Errorf("3 步能到，应归: %+v", near)
	}
}

// 隔着一个同样活跃的中间事件：远处的事件经中间事件串到源头（链式全部挂到最上游），路径可以超过 4 步
func TestGroup_ChainThroughActiveIntermediate(t *testing.T) {
	g := testGraph(
		[3]string{"a/service/s1", "a/service/s2", "calls"},
		[3]string{"a/service/s2", "a/service/s3", "calls"},
		[3]string{"a/service/s3", "a/service/s4", "calls"},
		[3]string{"a/service/s4", "a/service/s5", "calls"},
		[3]string{"a/service/s5", "a/pod/p", "selects"},
	)
	links := Group([]Candidate{
		cand("inc-root", "a/deployment/p", 0, 0.9, "a/pod/p"),
		cand("inc-far", "a/service/s1", time.Minute, 0.5, "a/service/s1"),
		cand("inc-near", "a/service/s3", time.Minute, 0.5, "a/service/s3"),
	}, g)
	if links["inc-near"].RootIncidentID != "inc-root" || links["inc-far"].RootIncidentID != "inc-root" {
		t.Fatalf("经中间事件都应挂到最上游: %+v", links)
	}
	if path := links["inc-far"].Path; len(path) != 6 || path[5].EntityKey != "a/pod/p" {
		t.Errorf("远处事件的路径应一直走到源头的成员: %+v", path)
	}
}

// 节点出问题：节点上的工作负载经 Pod → 运行在 → 节点，归到节点下
func TestGroup_WorkloadsUnderNode(t *testing.T) {
	g := testGraph([3]string{"a/pod/x-1", "_cluster/node/n1", "runs_on"}, [3]string{"b/pod/y-1", "_cluster/node/n1", "runs_on"})
	links := Group([]Candidate{
		cand("inc-node", "_cluster/node/n1", 0, 0.8, "_cluster/node/n1"),
		cand("inc-x", "a/deployment/x", time.Minute, 0.6, "a/pod/x-1"),
		cand("inc-y", "b/deployment/y", time.Minute, 0.6, "b/pod/y-1"),
	}, g)
	if links["inc-x"].RootIncidentID != "inc-node" || links["inc-y"].RootIncidentID != "inc-node" {
		t.Errorf("节点上的工作负载应归到节点下: %+v", links)
	}
}

// 一个事件有两个独立的上游起点：取开始最早的；同时开始取风险高的
func TestGroup_MultipleTopsPickEarliestThenRisk(t *testing.T) {
	g := testGraph([3]string{"a/service/api", "a/service/db", "calls"}, [3]string{"a/service/api", "a/service/cache", "calls"})
	links := Group([]Candidate{
		cand("inc-db", "a/service/db", time.Minute, 0.9, "a/service/db"),
		cand("inc-cache", "a/service/cache", 0, 0.5, "a/service/cache"),
		cand("inc-api", "a/service/api", 2*time.Minute, 0.6, "a/service/api"),
	}, g)
	if links["inc-api"].RootIncidentID != "inc-cache" {
		t.Errorf("应取开始最早的上游: %+v", links["inc-api"])
	}
	tie := Group([]Candidate{
		cand("inc-db", "a/service/db", 0, 0.9, "a/service/db"),
		cand("inc-cache", "a/service/cache", 0, 0.5, "a/service/cache"),
		cand("inc-api", "a/service/api", time.Minute, 0.6, "a/service/api"),
	}, g)
	if tie["inc-api"].RootIncidentID != "inc-db" {
		t.Errorf("同时开始应取风险高的: %+v", tie["inc-api"])
	}
}

// 互相调用的环：开始早的是源头，另一个归到它下面
func TestGroup_CycleEarliestIsRoot(t *testing.T) {
	g := testGraph([3]string{"a/service/x", "a/service/y", "calls"}, [3]string{"a/service/y", "a/service/x", "calls"})
	links := Group([]Candidate{
		cand("inc-x", "a/service/x", 0, 0.5, "a/service/x"),
		cand("inc-y", "a/service/y", time.Minute, 0.5, "a/service/y"),
	}, g)
	if links["inc-y"].RootIncidentID != "inc-x" || len(links) != 1 {
		t.Errorf("环里开始早的是源头: %+v", links)
	}
}

// 没有依赖关系的两个事件互不相干；没有依赖图时不归任何东西
func TestGroup_Unrelated(t *testing.T) {
	g := testGraph([3]string{"a/pod/p", "_cluster/node/n1", "runs_on"})
	if links := Group([]Candidate{
		cand("inc-a", "a/deployment/p", 0, 0.9, "a/pod/p"),
		cand("inc-b", "b/deployment/q", 0, 0.9, "b/pod/q"),
	}, g); len(links) != 0 {
		t.Errorf("没有依赖关系不应归: %+v", links)
	}
	if links := Group([]Candidate{cand("inc-a", "a/service/x", 0, 0.5, "a/service/x")}, nil); len(links) != 0 {
		t.Errorf("没有依赖图不应归: %+v", links)
	}
}

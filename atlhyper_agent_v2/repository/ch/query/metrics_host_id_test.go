package query

import (
	"context"
	"database/sql"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	agentmodel "AtlHyper/atlhyper_agent_v2/model"
	"AtlHyper/atlhyper_agent_v2/testutil/mock"
	"AtlHyper/model_v3/cluster"

	_ "modernc.org/sqlite"
)

// ──────────────────────────────────────────────────────────────
// 节点在 ClickHouse 里的主机标识（ResourceAttributes['net.host.name']）
// ──────────────────────────────────────────────────────────────
//
// 标识由 Collector 给 node-exporter 打的 instance 标签决定：当前是 K8s 节点名，
// 早期是节点 IP。按节点查询（历史曲线、单节点详情、单指标时序）必须用 ClickHouse
// 里实际出现的标识，否则 SQL 成功、0 行，页面只显示「暂无历史数据」。

func TestPickHostID(t *testing.T) {
	cases := []struct {
		name      string
		nodeName  string
		nodeIP    string
		activeIDs []string
		want      string
	}{
		{"标识是节点名", "node-2", "10.0.0.1", []string{"node-2", "node-3"}, "node-2"},
		{"标识是节点 IP", "node-2", "10.0.0.1", []string{"10.0.0.1", "10.0.0.2"}, "10.0.0.1"},
		{"名字与 IP 都出现时认名字", "node-2", "10.0.0.1", []string{"10.0.0.1", "node-2"}, "node-2"},
		{"都不在近期标识里时用节点名", "node-2", "10.0.0.1", []string{"other"}, "node-2"},
		{"近期标识为空（查询失败或没数据）时用节点名", "node-2", "10.0.0.1", nil, "node-2"},
		{"不知道节点 IP 时用节点名", "node-2", "", []string{"10.0.0.1"}, "node-2"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := pickHostID(c.nodeName, c.nodeIP, c.activeIDs); got != c.want {
				t.Errorf("pickHostID(%q, %q, %v) = %q, want %q", c.nodeName, c.nodeIP, c.activeIDs, got, c.want)
			}
		})
	}
}

// recordingCHClient 近期主机标识查询返回给定的标识，其余查询返回空结果并记下第一个参数。
type recordingCHClient struct {
	db      *sql.DB
	hostIDs []string

	mu   sync.Mutex
	args []string
}

func newRecordingCHClient(t *testing.T, hostIDs ...string) *recordingCHClient {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return &recordingCHClient{db: db, hostIDs: hostIDs}
}

func (c *recordingCHClient) Query(ctx context.Context, q string, args ...any) (*sql.Rows, error) {
	if strings.Contains(q, "SELECT DISTINCT ResourceAttributes['net.host.name']") {
		selects := make([]string, 0, len(c.hostIDs))
		for _, id := range c.hostIDs {
			selects = append(selects, "SELECT '"+id+"'")
		}
		return c.db.QueryContext(ctx, strings.Join(selects, " UNION ALL "))
	}
	if len(args) > 0 {
		if s, ok := args[0].(string); ok {
			c.mu.Lock()
			c.args = append(c.args, s)
			c.mu.Unlock()
		}
	}
	return c.db.QueryContext(ctx, "SELECT 1 WHERE 1 = 0")
}
func (c *recordingCHClient) QueryRow(ctx context.Context, q string, args ...any) *sql.Row {
	return c.db.QueryRowContext(ctx, "SELECT 1 WHERE 1 = 0")
}
func (c *recordingCHClient) Ping(ctx context.Context) error { return nil }
func (c *recordingCHClient) Close() error                   { return nil }

func (c *recordingCHClient) recordedArgs() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return slices.Clone(c.args)
}

func nodeRepoWith(nodes ...cluster.Node) *mock.NodeRepository {
	return &mock.NodeRepository{
		ListFn: func(context.Context, agentmodel.ListOptions) ([]cluster.Node, error) { return nodes, nil },
	}
}

func testNode(name, ip string) cluster.Node {
	return cluster.Node{
		Summary:   cluster.NodeSummary{Name: name},
		Addresses: cluster.NodeAddresses{InternalIP: ip},
	}
}

func TestGetNodeMetricsHistory_QueriesByHostIDInClickHouse(t *testing.T) {
	client := newRecordingCHClient(t, "node-2", "node-3")
	r := &metricsRepository{
		client:   client,
		nodeRepo: nodeRepoWith(testNode("node-2", "10.0.0.1"), testNode("node-3", "10.0.0.2")),
		ipMapTTL: time.Minute,
	}

	if _, err := r.GetNodeMetricsHistory(context.Background(), "node-2", 24*time.Hour); err != nil {
		t.Fatalf("GetNodeMetricsHistory: %v", err)
	}

	args := client.recordedArgs()
	if len(args) != 4 {
		t.Fatalf("应查询 cpu / memory / disk / temp 四个指标，实际 %d 次: %v", len(args), args)
	}
	for _, a := range args {
		if a != "node-2" {
			t.Errorf("ClickHouse 里的标识是节点名，查询却用了 %q（全部参数 %v）", a, args)
		}
	}
}

func TestGetNodeMetricsHistory_IPConvention(t *testing.T) {
	client := newRecordingCHClient(t, "10.0.0.1", "10.0.0.2")
	r := &metricsRepository{
		client:   client,
		nodeRepo: nodeRepoWith(testNode("node-2", "10.0.0.1"), testNode("node-3", "10.0.0.2")),
		ipMapTTL: time.Minute,
	}

	if _, err := r.GetNodeMetricsHistory(context.Background(), "node-2", 6*time.Hour); err != nil {
		t.Fatalf("GetNodeMetricsHistory: %v", err)
	}
	for _, a := range client.recordedArgs() {
		if a != "10.0.0.1" {
			t.Errorf("ClickHouse 里的标识是节点 IP，查询却用了 %q", a)
		}
	}
}

// 四个指标全部查询失败时必须报错：否则 Master 收到「成功 + 空序列」，页面只能显示「暂无历史数据」。
func TestGetNodeMetricsHistory_AllQueriesFail_ReturnsError(t *testing.T) {
	r := &metricsRepository{
		client:   newFailingCHClient(t),
		nodeRepo: nodeRepoWith(testNode("node-2", "10.0.0.1")),
		ipMapTTL: time.Minute,
	}

	if _, err := r.GetNodeMetricsHistory(context.Background(), "node-2", 24*time.Hour); err == nil {
		t.Fatal("四个指标查询全部失败，却返回了成功（错误被吞掉）")
	}
}

func TestUnmatchedNodes(t *testing.T) {
	nodes := []cluster.Node{testNode("node-2", "10.0.0.1"), testNode("node-3", "10.0.0.2")}
	cases := []struct {
		name    string
		hostIDs []string
		want    []string
	}{
		{"按节点名全部对上", []string{"node-2", "node-3"}, nil},
		{"按 IP 全部对上", []string{"10.0.0.1", "10.0.0.2"}, nil},
		{"部分节点没有主机指标", []string{"node-2"}, []string{"node-3"}},
		{"全部对不上（标识写法变了）", []string{"node-a", "node-b"}, []string{"node-2", "node-3"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := unmatchedNodes(nodes, c.hostIDs); !slices.Equal(got, c.want) {
				t.Errorf("unmatchedNodes(%v) = %v, want %v", c.hostIDs, got, c.want)
			}
		})
	}
}

package agentsdk

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/agentsdk/clustertoken"
	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/model_v3/cluster"
	"AtlHyper/model_v3/command"
)

// fakeDirectory 内存里的集群注册表：令牌哈希 -> 集群
type fakeDirectory struct {
	byHash map[string]*database.Cluster
	err    error
}

func (d *fakeDirectory) GetByTokenHash(_ context.Context, tokenHash string) (*database.Cluster, error) {
	if d.err != nil {
		return nil, d.err
	}
	return d.byHash[tokenHash], nil
}

// fakeProcessor 记录收到的快照与心跳
type fakeProcessor struct {
	snapshots  []string
	heartbeats []string
}

func (p *fakeProcessor) ProcessSnapshot(clusterID string, _ *cluster.ClusterSnapshot) error {
	p.snapshots = append(p.snapshots, clusterID)
	return nil
}

func (p *fakeProcessor) ProcessHeartbeat(clusterID string) error {
	p.heartbeats = append(p.heartbeats, clusterID)
	return nil
}

// ownershipBus 记得每条指令属于哪个集群，按 mq.Consumer 的约定核对归属
type ownershipBus struct {
	owner map[string]string
	acked map[string]*command.Result
}

func (b *ownershipBus) WaitCommand(context.Context, string, string, time.Duration) (*command.Command, error) {
	return nil, nil
}

func (b *ownershipBus) AckCommand(clusterID, cmdID string, result *command.Result) error {
	if owner, ok := b.owner[cmdID]; ok && clusterID != "" && owner != clusterID {
		return command.ErrNotOwnedByCluster
	}
	if b.acked == nil {
		b.acked = map[string]*command.Result{}
	}
	b.acked[cmdID] = result
	return nil
}

type authFixture struct {
	server    *Server
	processor *fakeProcessor
	bus       *ownershipBus
	tokenC1   string
}

func newAuthFixture(t *testing.T, mode AuthMode) *authFixture {
	t.Helper()
	issued, err := clustertoken.Generate()
	if err != nil {
		t.Fatal(err)
	}
	directory := &fakeDirectory{byHash: map[string]*database.Cluster{
		issued.Hash: {ClusterID: "c1", TokenHash: issued.Hash},
	}}
	processor := &fakeProcessor{}
	bus := &ownershipBus{owner: map[string]string{"cmd-c1": "c1", "cmd-c2": "c2"}}
	server := NewServer(Config{
		CommandTimeout: 50 * time.Millisecond,
		Bus:            bus,
		Processor:      processor,
		CmdRepo:        &fakeCmdRepo{},
		AuthMode:       mode,
		Clusters:       directory,
		AuthStatus:     NewAuthStatusTracker(),
	})
	return &authFixture{server: server, processor: processor, bus: bus, tokenC1: issued.Plaintext}
}

func (f *authFixture) do(method, target, token, claimedCluster string, body []byte) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, bytes.NewReader(body))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if claimedCluster != "" {
		req.Header.Set("X-Cluster-ID", claimedCluster)
	}
	rec := httptest.NewRecorder()
	f.server.routes().ServeHTTP(rec, req)
	return rec
}

func TestParseAuthMode(t *testing.T) {
	for _, valid := range []string{"off", "warn", "enforce"} {
		if mode, err := ParseAuthMode(valid); err != nil || string(mode) != valid {
			t.Errorf("ParseAuthMode(%q) = %q, %v", valid, mode, err)
		}
	}
	if _, err := ParseAuthMode("strict"); err == nil {
		t.Error("未知的模式应报错，不能悄悄当成某一档")
	}
}

func TestHeartbeat_ByModeAndToken(t *testing.T) {
	tests := []struct {
		name        string
		mode        AuthMode
		token       string // "valid" / "invalid" / ""
		claimed     string
		wantStatus  int
		wantCluster string // 期望记到哪个集群名下；空 = 不应被处理
	}{
		{name: "enforce：有效令牌，自报一致", mode: AuthModeEnforce, token: "valid", claimed: "c1", wantStatus: 200, wantCluster: "c1"},
		{name: "enforce：有效令牌，没自报集群也行", mode: AuthModeEnforce, token: "valid", wantStatus: 200, wantCluster: "c1"},
		{name: "enforce：没带令牌", mode: AuthModeEnforce, claimed: "c1", wantStatus: 401},
		{name: "enforce：令牌无效", mode: AuthModeEnforce, token: "invalid", claimed: "c1", wantStatus: 401},
		{name: "enforce：有效令牌却自报另一个集群", mode: AuthModeEnforce, token: "valid", claimed: "c2", wantStatus: 403},
		{name: "warn：没带令牌照常接收", mode: AuthModeWarn, claimed: "c9", wantStatus: 200, wantCluster: "c9"},
		{name: "warn：令牌无效照常接收", mode: AuthModeWarn, token: "invalid", claimed: "c9", wantStatus: 200, wantCluster: "c9"},
		{name: "warn：有效令牌以令牌的集群为准", mode: AuthModeWarn, token: "valid", wantStatus: 200, wantCluster: "c1"},
		{name: "warn：有效令牌却自报另一个集群", mode: AuthModeWarn, token: "valid", claimed: "c2", wantStatus: 403},
		{name: "off：完全不检查", mode: AuthModeOff, token: "invalid", claimed: "c9", wantStatus: 200, wantCluster: "c9"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newAuthFixture(t, tt.mode)
			token := tt.token
			switch token {
			case "valid":
				token = f.tokenC1
			case "invalid":
				token = "atlk_not-a-real-token"
			}

			rec := f.do(http.MethodPost, "/agent/heartbeat", token, tt.claimed, nil)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d（%s）", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if tt.wantCluster == "" {
				if len(f.processor.heartbeats) != 0 {
					t.Errorf("被拒绝的请求不应被处理，实际处理了 %v", f.processor.heartbeats)
				}
				return
			}
			if len(f.processor.heartbeats) != 1 || f.processor.heartbeats[0] != tt.wantCluster {
				t.Errorf("心跳应记到 %q 名下，实际 %v", tt.wantCluster, f.processor.heartbeats)
			}
		})
	}
}

// 管理页要显示「这个集群的 Agent 是否已带令牌」，由每次心跳与快照更新。
func TestHeartbeat_RecordsWhetherAgentIsAuthenticated(t *testing.T) {
	f := newAuthFixture(t, AuthModeWarn)

	f.do(http.MethodPost, "/agent/heartbeat", f.tokenC1, "c1", nil)
	f.do(http.MethodPost, "/agent/heartbeat", "", "c9", nil)

	if authenticated, known := f.server.authStatus.IsAuthenticated("c1"); !known || !authenticated {
		t.Error("带有效令牌的集群应标记为已认证")
	}
	if authenticated, known := f.server.authStatus.IsAuthenticated("c9"); !known || authenticated {
		t.Errorf("没带令牌的集群应标记为未认证，实际 known=%v authenticated=%v", known, authenticated)
	}
	if _, known := f.server.authStatus.IsAuthenticated("never-seen"); known {
		t.Error("没上报过的集群不应有认证状态")
	}
}

func gzipJSON(t *testing.T, value any) []byte {
	t.Helper()
	var buf bytes.Buffer
	writer := gzip.NewWriter(&buf)
	if err := json.NewEncoder(writer).Encode(value); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func (f *authFixture) pushSnapshot(t *testing.T, token, claimedCluster, bodyCluster string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/agent/snapshot",
		bytes.NewReader(gzipJSON(t, cluster.ClusterSnapshot{ClusterID: bodyCluster})))
	req.Header.Set("Content-Encoding", "gzip")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if claimedCluster != "" {
		req.Header.Set("X-Cluster-ID", claimedCluster)
	}
	rec := httptest.NewRecorder()
	f.server.routes().ServeHTTP(rec, req)
	return rec
}

func TestSnapshot_StoredUnderAuthenticatedCluster(t *testing.T) {
	f := newAuthFixture(t, AuthModeEnforce)

	if rec := f.pushSnapshot(t, f.tokenC1, "", "c1"); rec.Code != http.StatusOK {
		t.Fatalf("status = %d（%s）", rec.Code, rec.Body.String())
	}
	if len(f.processor.snapshots) != 1 || f.processor.snapshots[0] != "c1" {
		t.Errorf("快照应记到 c1 名下，实际 %v", f.processor.snapshots)
	}
}

// 请求头没自报、请求体里写的是另一个集群：同样不能越过令牌的身份。
func TestSnapshot_BodyClaimsAnotherCluster(t *testing.T) {
	f := newAuthFixture(t, AuthModeEnforce)

	rec := f.pushSnapshot(t, f.tokenC1, "", "c2")

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
	if len(f.processor.snapshots) != 0 {
		t.Errorf("被拒绝的快照不应入库，实际 %v", f.processor.snapshots)
	}
}

func TestSnapshot_TooLarge(t *testing.T) {
	f := newAuthFixture(t, AuthModeOff)
	f.server.maxSnapshotBytes = 1024

	// 压缩后很小、解压后远超上限
	huge := cluster.ClusterSnapshot{ClusterID: "c1", Pods: make([]cluster.Pod, 0)}
	payload := map[string]any{"clusterId": huge.ClusterID, "padding": strings.Repeat("a", 1<<20)}
	req := httptest.NewRequest(http.MethodPost, "/agent/snapshot", bytes.NewReader(gzipJSON(t, payload)))
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("X-Cluster-ID", "c1")
	rec := httptest.NewRecorder()
	f.server.routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413", rec.Code)
	}
	if len(f.processor.snapshots) != 0 {
		t.Error("超限的快照不应入库")
	}
}

func resultBody(t *testing.T, commandID string) []byte {
	t.Helper()
	body, err := json.Marshal(ResultRequest{CommandID: commandID, Success: true, Output: "ok"})
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func TestResult_OnlyOwningClusterMayReport(t *testing.T) {
	f := newAuthFixture(t, AuthModeEnforce)

	own := f.do(http.MethodPost, "/agent/result", f.tokenC1, "c1", resultBody(t, "cmd-c1"))
	if own.Code != http.StatusOK {
		t.Fatalf("回报自己集群的指令 status = %d（%s）", own.Code, own.Body.String())
	}
	if _, ok := f.bus.acked["cmd-c1"]; !ok {
		t.Error("自己集群的指令结果应被接受")
	}

	other := f.do(http.MethodPost, "/agent/result", f.tokenC1, "c1", resultBody(t, "cmd-c2"))
	if other.Code != http.StatusForbidden {
		t.Fatalf("回报别的集群的指令 status = %d, want 403", other.Code)
	}
	if _, ok := f.bus.acked["cmd-c2"]; ok {
		t.Error("别的集群的指令结果不应被写入")
	}
}

func TestCommands_PolledForAuthenticatedCluster(t *testing.T) {
	f := newAuthFixture(t, AuthModeEnforce)

	if rec := f.do(http.MethodGet, "/agent/commands?cluster_id=c1&topic=ops", f.tokenC1, "", nil); rec.Code != http.StatusOK {
		t.Fatalf("status = %d（%s）", rec.Code, rec.Body.String())
	}
	if rec := f.do(http.MethodGet, "/agent/commands?cluster_id=c2&topic=ops", f.tokenC1, "", nil); rec.Code != http.StatusForbidden {
		t.Fatalf("拉取别的集群的指令 status = %d, want 403", rec.Code)
	}
	if rec := f.do(http.MethodGet, "/agent/commands?cluster_id=c1&topic=ops", "", "", nil); rec.Code != http.StatusUnauthorized {
		t.Fatalf("没带令牌 status = %d, want 401", rec.Code)
	}
}

// 注册表查不了时：强制模式拒绝（不能因为查不了就放行），告警模式照常接收。
func TestDirectoryFailure(t *testing.T) {
	enforce := newAuthFixture(t, AuthModeEnforce)
	enforce.server.clusters = &fakeDirectory{err: errors.New("database is locked")}
	if rec := enforce.do(http.MethodPost, "/agent/heartbeat", enforce.tokenC1, "c1", nil); rec.Code != http.StatusServiceUnavailable {
		t.Errorf("enforce 下注册表不可用 status = %d, want 503", rec.Code)
	}

	warn := newAuthFixture(t, AuthModeWarn)
	warn.server.clusters = &fakeDirectory{err: errors.New("database is locked")}
	if rec := warn.do(http.MethodPost, "/agent/heartbeat", warn.tokenC1, "c1", nil); rec.Code != http.StatusOK {
		t.Errorf("warn 下注册表不可用 status = %d, want 200", rec.Code)
	}
}

func TestHealth_NeedsNoToken(t *testing.T) {
	f := newAuthFixture(t, AuthModeEnforce)
	if rec := f.do(http.MethodGet, "/health", "", "", nil); rec.Code != http.StatusOK {
		t.Errorf("健康检查 status = %d, want 200", rec.Code)
	}
}

func TestWarnLimiter_OncePerWindowPerKey(t *testing.T) {
	current := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	limiter := newWarnLimiter(time.Minute, func() time.Time { return current })

	if !limiter.allow("c1|no-token") {
		t.Error("第一次应放行")
	}
	if limiter.allow("c1|no-token") {
		t.Error("同一窗口内的第二次应被压掉")
	}
	if !limiter.allow("c2|no-token") {
		t.Error("另一个集群不受影响")
	}
	current = current.Add(61 * time.Second)
	if !limiter.allow("c1|no-token") {
		t.Error("窗口过后应再次放行")
	}
}

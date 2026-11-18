package gateway

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"AtlHyper/model_v3/cluster"
	"AtlHyper/model_v3/command"
)

// 测试用的令牌：明显是占位值，不是真实令牌
const placeholderMasterToken = "atlk_placeholder-not-a-real-token"

// masterChannels 四条通道各发一次请求
func masterChannels(gw MasterGateway) map[string]func() error {
	ctx := context.Background()
	return map[string]func() error{
		"推快照":  func() error { return gw.PushSnapshot(ctx, &cluster.ClusterSnapshot{ClusterID: "test-cluster"}) },
		"拉指令":  func() error { _, err := gw.PollCommands(ctx, "ops"); return err },
		"回报结果": func() error { return gw.ReportResult(ctx, &command.Result{CommandID: "cmd-1"}) },
		"心跳":   func() error { return gw.Heartbeat(ctx) },
	}
}

// recordingMaster 记下每个路径收到的 Authorization 头，并统一回一个状态码
type recordingMaster struct {
	mu            sync.Mutex
	authorization map[string]string
	seen          map[string]bool
}

func newRecordingMaster(t *testing.T, status int, body string) (*recordingMaster, *httptest.Server) {
	t.Helper()
	master := &recordingMaster{authorization: map[string]string{}, seen: map[string]bool{}}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		master.mu.Lock()
		master.authorization[r.URL.Path] = r.Header.Get("Authorization")
		master.seen[r.URL.Path] = true
		master.mu.Unlock()
		w.WriteHeader(status)
		w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	return master, server
}

var agentPaths = []string{"/agent/snapshot", "/agent/commands", "/agent/result", "/agent/heartbeat"}

func TestMasterGateway_BearerTokenOnEveryChannel(t *testing.T) {
	cases := []struct {
		name       string
		token      string
		wantHeader string
	}{
		{"配置了令牌", placeholderMasterToken, "Bearer " + placeholderMasterToken},
		{"没配置令牌时不带认证头", "", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			master, server := newRecordingMaster(t, http.StatusNoContent, "")
			gw := NewMasterGateway(server.URL, "test-cluster", c.token, 5*time.Second)

			// 204 对拉指令是「没有指令」，对其余通道是意外状态码；这里只关心请求头
			for _, send := range masterChannels(gw) {
				_ = send()
			}

			for _, path := range agentPaths {
				if !master.seen[path] {
					t.Errorf("%s 没有收到请求", path)
					continue
				}
				if got := master.authorization[path]; got != c.wantHeader {
					t.Errorf("%s 的 Authorization = %q, want %q", path, got, c.wantHeader)
				}
			}
		})
	}
}

// 集群 ID 与 topic 作为查询参数发送时必须转义：特殊字符不得改变请求的含义。
func TestPollCommands_QueryParametersAreEscaped(t *testing.T) {
	cases := []struct {
		name      string
		clusterID string
		topic     string
	}{
		{"普通取值", "prod-1", "ops"},
		{"带空格与分隔符", "a b&topic=ai", "ops"},
		{"带井号与百分号", "c#1%2F", "ai"},
		{"非 ASCII", "生产", "ops"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var gotCluster, gotTopic string
			var gotParams int
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				query := r.URL.Query()
				gotCluster, gotTopic, gotParams = query.Get("cluster_id"), query.Get("topic"), len(query)
				w.WriteHeader(http.StatusNoContent)
			}))
			defer server.Close()

			gw := NewMasterGateway(server.URL, c.clusterID, "", 5*time.Second)
			if _, err := gw.PollCommands(context.Background(), c.topic); err != nil {
				t.Fatal(err)
			}
			if gotCluster != c.clusterID || gotTopic != c.topic || gotParams != 2 {
				t.Errorf("Master 收到 cluster_id=%q topic=%q（共 %d 个参数），want %q / %q / 2",
					gotCluster, gotTopic, gotParams, c.clusterID, c.topic)
			}
		})
	}
}

// Master 以 401 / 403 拒绝时，四条通道返回的错误都能被认出是身份问题，且不带出令牌。
func TestMasterGateway_IdentityRejection(t *testing.T) {
	cases := []struct {
		name         string
		status       int
		body         string
		wantRejected bool
	}{
		{"401 没带有效令牌", http.StatusUnauthorized, "valid cluster token required\n", true},
		{"403 令牌与集群不符", http.StatusForbidden, "token does not belong to the claimed cluster\n", true},
		{"500 不是身份问题", http.StatusInternalServerError, "Internal error\n", false},
		{"503 不是身份问题", http.StatusServiceUnavailable, "cluster registry unavailable\n", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, server := newRecordingMaster(t, c.status, c.body)
			gw := NewMasterGateway(server.URL, "test-cluster", placeholderMasterToken, 5*time.Second)

			for channel, send := range masterChannels(gw) {
				err := send()
				if err == nil {
					t.Errorf("%s：期望报错", channel)
					continue
				}
				if got := errors.Is(err, ErrMasterRejectedIdentity); got != c.wantRejected {
					t.Errorf("%s：errors.Is(ErrMasterRejectedIdentity) = %v, want %v（%v）", channel, got, c.wantRejected, err)
				}
				if strings.Contains(err.Error(), placeholderMasterToken) {
					t.Errorf("%s：错误信息里不得出现令牌：%v", channel, err)
				}
				if c.wantRejected && !strings.Contains(err.Error(), strings.TrimSpace(c.body)) {
					t.Errorf("%s：错误信息应带上 Master 给出的原因：%v", channel, err)
				}
			}
		})
	}
}

// Master 返回的响应体只截取开头一小段写进错误信息，不把整段内容带进日志。
func TestMasterGateway_RejectionBodyIsTruncated(t *testing.T) {
	_, server := newRecordingMaster(t, http.StatusUnauthorized, strings.Repeat("x", 4096))
	gw := NewMasterGateway(server.URL, "test-cluster", placeholderMasterToken, 5*time.Second)

	err := gw.Heartbeat(context.Background())
	if err == nil || len(err.Error()) > 1024 {
		t.Errorf("错误信息长度 = %d，应被截断", len(err.Error()))
	}
}

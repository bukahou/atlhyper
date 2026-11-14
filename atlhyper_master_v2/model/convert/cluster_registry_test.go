package convert

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/model_v3/agent"
)

func TestClusterRegistryItem(t *testing.T) {
	// 带时区的时间：输出必须统一换成 UTC
	tokyo := time.FixedZone("JST", 9*3600)
	issuedAt := time.Date(2026, 11, 14, 19, 0, 0, 0, tokyo)
	heartbeat := time.Date(2026, 11, 14, 19, 5, 0, 0, tokyo)
	snapshot := time.Date(2026, 11, 14, 19, 5, 7, 0, tokyo)

	withToken := &database.Cluster{
		ClusterID: "c1", Name: "生产", Description: "东京",
		TokenHash: "not-a-real-hash", TokenPrefix: "atlk_Ab3", TokenIssuedAt: &issuedAt,
	}
	withoutToken := &database.Cluster{ClusterID: "c2", Name: "预发"}

	cases := []struct {
		name          string
		registered    *database.Cluster
		presence      *agent.AgentInfo
		authenticated bool
		want          model.ClusterRegistryItem
	}{
		{
			name:          "已登记、已签发、在线、已认证",
			registered:    withToken,
			presence:      &agent.AgentInfo{ClusterID: "c1", Status: agent.StatusOnline, LastHeartbeat: heartbeat},
			authenticated: true,
			want: model.ClusterRegistryItem{
				ClusterID: "c1", Name: "生产", Description: "东京", Registered: true,
				TokenIssued: true, TokenPrefix: "atlk_Ab3", TokenIssuedAt: "2026-11-14T10:00:00Z",
				Status: model.ClusterStatusOnline, LastSeen: "2026-11-14T10:05:00Z", Authenticated: true,
			},
		},
		{
			name:       "已登记但从未上报",
			registered: withoutToken,
			want: model.ClusterRegistryItem{
				ClusterID: "c2", Name: "预发", Registered: true, Status: model.ClusterStatusNever,
			},
		},
		{
			name:       "已登记、已离线",
			registered: withoutToken,
			presence:   &agent.AgentInfo{ClusterID: "c2", Status: agent.StatusOffline, LastHeartbeat: heartbeat},
			want: model.ClusterRegistryItem{
				ClusterID: "c2", Name: "预发", Registered: true,
				Status: model.ClusterStatusOffline, LastSeen: "2026-11-14T10:05:00Z",
			},
		},
		{
			name:     "正在上报但未登记",
			presence: &agent.AgentInfo{ClusterID: "c3", Status: agent.StatusOnline, LastHeartbeat: heartbeat},
			want: model.ClusterRegistryItem{
				ClusterID: "c3", Registered: false,
				Status: model.ClusterStatusOnline, LastSeen: "2026-11-14T10:05:00Z",
			},
		},
		{
			name:     "最近一次出现取心跳与快照里较晚的那个",
			presence: &agent.AgentInfo{ClusterID: "c3", Status: agent.StatusOnline, LastHeartbeat: heartbeat, LastSnapshot: snapshot},
			want: model.ClusterRegistryItem{
				ClusterID: "c3", Status: model.ClusterStatusOnline, LastSeen: "2026-11-14T10:05:07Z",
			},
		},
		{
			name:     "状态取值以外的一律按离线",
			presence: &agent.AgentInfo{ClusterID: "c3", Status: "unknown", LastHeartbeat: heartbeat},
			want: model.ClusterRegistryItem{
				ClusterID: "c3", Status: model.ClusterStatusOffline, LastSeen: "2026-11-14T10:05:00Z",
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := ClusterRegistryItem(c.registered, c.presence, c.authenticated)
			if got != c.want {
				t.Errorf("\n got %+v\nwant %+v", got, c.want)
			}
		})
	}
}

// 响应的 JSON 字段名与设计文档逐字一致，且令牌哈希不出现在其中。
func TestClusterRegistryItem_JSONShape(t *testing.T) {
	issuedAt := time.Date(2026, 11, 14, 10, 0, 0, 0, time.UTC)
	item := ClusterRegistryItem(
		&database.Cluster{ClusterID: "c1", Name: "生产", TokenHash: "not-a-real-hash", TokenPrefix: "atlk_Ab3", TokenIssuedAt: &issuedAt},
		nil, false,
	)
	raw, err := json.Marshal(item)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]any
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatal(err)
	}
	wantKeys := []string{
		"clusterId", "name", "description", "registered", "tokenIssued", "tokenPrefix",
		"tokenIssuedAt", "status", "lastSeen", "authenticated",
	}
	if len(fields) != len(wantKeys) {
		t.Errorf("字段数 = %d, want %d：%s", len(fields), len(wantKeys), raw)
	}
	for _, key := range wantKeys {
		if _, ok := fields[key]; !ok {
			t.Errorf("缺少字段 %q：%s", key, raw)
		}
	}
	if strings.Contains(string(raw), "not-a-real-hash") {
		t.Errorf("令牌哈希不得出现在响应里：%s", raw)
	}
}

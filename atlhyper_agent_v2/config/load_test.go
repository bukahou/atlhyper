package config

import (
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

func clearAgentEnv(t *testing.T) {
	t.Helper()
	for _, kv := range os.Environ() {
		key, _, _ := strings.Cut(kv, "=")
		if strings.HasPrefix(key, "AGENT_") || key == "KUBECONFIG" {
			t.Setenv(key, "")
			os.Unsetenv(key)
		}
	}
}

// placeholderToken 测试用占位令牌（⛔ 不是真实令牌）
const placeholderToken = "atlk_placeholder-not-a-real-token"

// setRequiredAgentEnv 只填必填项（集群令牌），其余保持未设置
func setRequiredAgentEnv(t *testing.T) {
	t.Helper()
	t.Setenv("AGENT_MASTER_TOKEN", placeholderToken)
}

func mustLoad(t *testing.T) AppConfig {
	t.Helper()
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return cfg
}

// 只配必填项（集群令牌）时的默认值（变量名与默认值改造前后都不变）
func TestLoad_Defaults(t *testing.T) {
	clearAgentEnv(t)
	setRequiredAgentEnv(t)
	c := mustLoad(t)
	want := AppConfig{
		Log:        LogConfig{Level: "info", Format: "text"},
		Agent:      AgentConfig{ClusterID: "", HealthAddr: ":8082"},
		Master:     MasterConfig{URL: "http://localhost:8081", Token: placeholderToken},
		Kubernetes: KubernetesConfig{KubeConfig: ""},
		Scheduler: SchedulerConfig{
			SnapshotInterval: 10 * time.Second, CommandPollInterval: 100 * time.Millisecond,
			HeartbeatInterval: 15 * time.Second, OTelCacheTTL: 10 * time.Second, OTelDashboardTTL: 30 * time.Second,
		},
		Timeout: TimeoutConfig{
			HTTPClient: 90 * time.Second, SnapshotCollect: 30 * time.Second, CommandPoll: 60 * time.Second,
			CommandExec: 60 * time.Second, CommandReport: 10 * time.Second, Heartbeat: 10 * time.Second,
		},
		ClickHouse: ClickHouseConfig{Endpoint: "", Database: "atlhyper", Timeout: 10 * time.Second},
	}
	if !reflect.DeepEqual(c, want) {
		t.Errorf("默认配置:\n got %+v\nwant %+v", c, want)
	}
}

// 每个变量名都接上了
func TestLoad_EnvOverrides(t *testing.T) {
	clearAgentEnv(t)
	for k, v := range map[string]string{
		"AGENT_LOG_LEVEL": "debug", "AGENT_LOG_FORMAT": "json",
		"AGENT_CLUSTER_ID": "c1", "AGENT_HEALTH_ADDR": ":9000",
		"AGENT_MASTER_URL": "http://master.example.com:8081", "AGENT_MASTER_TOKEN": " atlk_placeholder-not-a-real-token\n",
		"AGENT_KUBECONFIG":        "/k/config",
		"AGENT_SNAPSHOT_INTERVAL": "1s", "AGENT_COMMAND_POLL_INTERVAL": "2s", "AGENT_HEARTBEAT_INTERVAL": "3s",
		"AGENT_OTEL_CACHE_TTL": "4s", "AGENT_OTEL_DASHBOARD_TTL": "5s",
		"AGENT_TIMEOUT_HTTP_CLIENT": "6s", "AGENT_TIMEOUT_SNAPSHOT_COLLECT": "7s", "AGENT_TIMEOUT_COMMAND_POLL": "8s",
		"AGENT_TIMEOUT_COMMAND_EXEC": "9s", "AGENT_TIMEOUT_COMMAND_REPORT": "10s", "AGENT_TIMEOUT_HEARTBEAT": "11s",
		"AGENT_CLICKHOUSE_ENDPOINT": "clickhouse://ch.example.com:9000", "AGENT_CLICKHOUSE_DATABASE": "otel",
		"AGENT_CLICKHOUSE_TIMEOUT": "12s",
	} {
		t.Setenv(k, v)
	}
	c := mustLoad(t)
	want := AppConfig{
		Log:        LogConfig{Level: "debug", Format: "json"},
		Agent:      AgentConfig{ClusterID: "c1", HealthAddr: ":9000"},
		Master:     MasterConfig{URL: "http://master.example.com:8081", Token: "atlk_placeholder-not-a-real-token"},
		Kubernetes: KubernetesConfig{KubeConfig: "/k/config"},
		Scheduler:  SchedulerConfig{SnapshotInterval: time.Second, CommandPollInterval: 2 * time.Second, HeartbeatInterval: 3 * time.Second, OTelCacheTTL: 4 * time.Second, OTelDashboardTTL: 5 * time.Second},
		Timeout:    TimeoutConfig{HTTPClient: 6 * time.Second, SnapshotCollect: 7 * time.Second, CommandPoll: 8 * time.Second, CommandExec: 9 * time.Second, CommandReport: 10 * time.Second, Heartbeat: 11 * time.Second},
		ClickHouse: ClickHouseConfig{Endpoint: "clickhouse://ch.example.com:9000", Database: "otel", Timeout: 12 * time.Second},
	}
	if !reflect.DeepEqual(c, want) {
		t.Errorf("覆盖后:\n got %+v\nwant %+v", c, want)
	}
}

// kubeconfig：AGENT_KUBECONFIG 优先，其次标准的 KUBECONFIG
func TestLoad_KubeConfigFallback(t *testing.T) {
	clearAgentEnv(t)
	setRequiredAgentEnv(t)
	t.Setenv("KUBECONFIG", "/std/config")
	if got := mustLoad(t).Kubernetes.KubeConfig; got != "/std/config" {
		t.Errorf("只有 KUBECONFIG 时 = %q", got)
	}
	t.Setenv("AGENT_KUBECONFIG", "/agent/config")
	if got := mustLoad(t).Kubernetes.KubeConfig; got != "/agent/config" {
		t.Errorf("两者都有时 = %q, want AGENT_KUBECONFIG", got)
	}
}

// 格式写错：返回错误并指出变量名，⛔ 不静默用默认值
func TestLoad_InvalidDurationFails(t *testing.T) {
	clearAgentEnv(t)
	setRequiredAgentEnv(t)
	t.Setenv("AGENT_SNAPSHOT_INTERVAL", "10")
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "AGENT_SNAPSHOT_INTERVAL") {
		t.Fatalf("应返回指出 AGENT_SNAPSHOT_INTERVAL 的错误，实际 %v", err)
	}
}

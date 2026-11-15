package config

import "testing"

// Agent 入口的两项配置：默认以告警模式运行（升级后先观察、不拒绝任何 Agent），快照上限 256 MB。
func TestDefaults_AgentEntry(t *testing.T) {
	if got := defaultStrings["MASTER_AGENT_AUTH_MODE"]; got != "warn" {
		t.Errorf("MASTER_AGENT_AUTH_MODE 默认应为 warn，得到 %q", got)
	}
	if got := defaultInts["MASTER_AGENTSDK_MAX_SNAPSHOT_MB"]; got != 256 {
		t.Errorf("MASTER_AGENTSDK_MAX_SNAPSHOT_MB 默认应为 256，得到 %d", got)
	}
}

// 环境变量覆盖默认值；认证模式原样读出，是否合法由启动时的校验判断。
func TestLoad_AgentEntryFromEnv(t *testing.T) {
	t.Setenv("MASTER_AGENT_AUTH_MODE", "enforce")
	t.Setenv("MASTER_AGENTSDK_MAX_SNAPSHOT_MB", "64")

	if got := getString("MASTER_AGENT_AUTH_MODE"); got != "enforce" {
		t.Errorf("MASTER_AGENT_AUTH_MODE = %q, want enforce", got)
	}
	if got := getInt("MASTER_AGENTSDK_MAX_SNAPSHOT_MB"); got != 64 {
		t.Errorf("MASTER_AGENTSDK_MAX_SNAPSHOT_MB = %d, want 64", got)
	}
}

// 对外访问地址不是凭据，默认为空（通知里不带页面链接）；可由环境变量给出。
func TestLoad_PublicURL(t *testing.T) {
	if got, defined := defaultStrings["MASTER_PUBLIC_URL"]; !defined || got != "" {
		t.Errorf("MASTER_PUBLIC_URL 应有空的默认值，得到 (%q, %v)", got, defined)
	}
	t.Setenv("MASTER_PUBLIC_URL", "https://atlhyper.example.com")
	if got := getString("MASTER_PUBLIC_URL"); got != "https://atlhyper.example.com" {
		t.Errorf("MASTER_PUBLIC_URL = %q", got)
	}
}

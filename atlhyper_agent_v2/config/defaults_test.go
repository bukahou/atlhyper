package config

import "testing"

// D3（2026-09-11）：Dashboard 列表缓存 TTL 从硬编码 30s 提成可配，
// dev 环境可放宽到分钟级以减少对共用 ClickHouse 的查询压力。默认值必须与旧行为一致。
func TestDefaults_OTelDashboardTTL(t *testing.T) {
	if got := defaultDurations["AGENT_OTEL_DASHBOARD_TTL"]; got != "30s" {
		t.Fatalf("AGENT_OTEL_DASHBOARD_TTL 默认应为 30s（与旧硬编码一致），得到 %q", got)
	}
}

// 集群令牌没有默认值：不配置时 Agent 不带认证头，配置与否由部署决定。
func TestDefaults_MasterTokenIsEmpty(t *testing.T) {
	got, defined := defaultStrings["AGENT_MASTER_TOKEN"]
	if !defined || got != "" {
		t.Fatalf("AGENT_MASTER_TOKEN 应已登记且默认为空，得到 defined=%v value=%q", defined, got)
	}
}

func TestLoadMasterToken(t *testing.T) {
	cases := []struct {
		name string
		env  string
		want string
	}{
		{"未设置", "", ""},
		{"正常取值", "atlk_placeholder-not-a-real-token", "atlk_placeholder-not-a-real-token"},
		// 从文件或 Secret 注入时常带换行；带着换行的值放不进请求头
		{"去掉首尾空白与换行", "  atlk_placeholder-not-a-real-token\n", "atlk_placeholder-not-a-real-token"},
		{"只有空白等同未设置", " \n", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Setenv("AGENT_MASTER_TOKEN", c.env)
			if got := loadMasterToken(); got != c.want {
				t.Errorf("loadMasterToken() = %q, want %q", got, c.want)
			}
		})
	}
}

// 启动日志只说明令牌有没有配置，任何情况下都不带出它的内容。
func TestDescribeSecretPresence(t *testing.T) {
	const token = "atlk_placeholder-not-a-real-token"
	if got := describeSecretPresence(token); got != "已配置" {
		t.Errorf("有值时 = %q, want 已配置", got)
	}
	if got := describeSecretPresence(""); got != "未配置" {
		t.Errorf("空值时 = %q, want 未配置", got)
	}
}

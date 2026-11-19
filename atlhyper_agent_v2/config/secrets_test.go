package config

import (
	"strings"
	"testing"
)

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
			clearAgentEnv(t)
			t.Setenv("AGENT_MASTER_TOKEN", c.env)
			if got := mustLoad(t).Master.Token; got != c.want {
				t.Errorf("Master.Token = %q, want %q", got, c.want)
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

// ClickHouse 地址里的密码不进日志
func TestRedactedEndpoint(t *testing.T) {
	got := RedactedEndpoint("clickhouse://reader:placeholder-not-real@ch.example.com:9000/otel")
	if strings.Contains(got, "placeholder-not-real") || !strings.Contains(got, "reader") || !strings.Contains(got, "ch.example.com") {
		t.Errorf("RedactedEndpoint = %q", got)
	}
	if got := RedactedEndpoint("ch.example.com:9000"); strings.Contains(got, "@") {
		t.Errorf("不带凭据的地址: %q", got)
	}
}

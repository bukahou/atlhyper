package config

import "testing"

// D3（2026-09-11）：Dashboard 列表缓存 TTL 从硬编码 30s 提成可配，
// dev 环境可放宽到分钟级以减少对共用 ClickHouse 的查询压力。默认值必须与旧行为一致。
func TestDefaults_OTelDashboardTTL(t *testing.T) {
	if got := defaultDurations["AGENT_OTEL_DASHBOARD_TTL"]; got != "30s" {
		t.Fatalf("AGENT_OTEL_DASHBOARD_TTL 默认应为 30s（与旧硬编码一致），得到 %q", got)
	}
}

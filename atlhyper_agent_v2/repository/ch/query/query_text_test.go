package query

import (
	"strings"
	"testing"
	"time"
)

// 2026-09-11 ClickHouse 负载排查：ClickHouseClient 返回 *sql.Row 无法 mock，
// 故把 SQL 文本抽成常量/函数在这里守住 —— 这两条曾是全库扫描量的第一与第三名。

// fillSystemInfo 的 uname 查询原本没有任何时间过滤：ORDER BY TimeUnix DESC LIMIT 1
// 无法用主键（TimeUnix 排在 Map 列之后），于是扫全部 7 天分区（实测 128,789 行 / 45 MiB，
// 每 30s × 每节点 × 每 agent）。uname 每 15s 上报一次，15 分钟窗口必有值。
func TestUnameQuery_HasTimeWindow(t *testing.T) {
	if !strings.Contains(unameQuery, "TimeUnix >= now() - INTERVAL 15 MINUTE") {
		t.Fatalf("uname 查询必须带 15 分钟时间窗，否则扫全部分区:\n%s", unameQuery)
	}
	if !strings.Contains(unameQuery, "node_uname_info") || !strings.Contains(unameQuery, "LIMIT 1") {
		t.Fatalf("uname 查询主体不应改变:\n%s", unameQuery)
	}
}

// 新鲜度原本对每张表 SELECT max(时间列) WHERE 时间列 > now()-24h：分区裁剪只到「天」，
// 分区内主键不裁 ⇒ gauge 表每次扫 ≈910 万行 / 70 MiB，每 10s × 每 agent。
// system.parts 的 max_time 是 MergeTree 为每个 part 维护的元数据，读 20 来行就有答案。
func TestFreshnessQuery_ReadsSystemPartsNotTheTable(t *testing.T) {
	q := freshnessQuery("otel_traces")
	for _, must := range []string{"system.parts", "max(max_time)", "active", "'otel_traces'"} {
		if !strings.Contains(q, must) {
			t.Errorf("新鲜度查询缺少 %q:\n%s", must, q)
		}
	}
	if strings.Contains(q, "FROM otel_traces") {
		t.Errorf("新鲜度查询不应再扫业务表:\n%s", q)
	}
}

// system.parts 上空表/无 part 时 max(max_time) 是 1970-01-01 而非 Go 零值；
// Master 只把零值判为 absent，1970 会被当成「48 年前有数据」。
func TestNormalizeFreshnessTime(t *testing.T) {
	epoch := time.Unix(0, 0).UTC()
	real := time.Date(2026, 9, 11, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		in   time.Time
		want time.Time
	}{
		{"零值保持零值", time.Time{}, time.Time{}},
		{"1970 视为无数据", epoch, time.Time{}},
		{"正常时间原样返回", real, real},
	}
	for _, tc := range tests {
		if got := normalizeFreshnessTime(tc.in); !got.Equal(tc.want) {
			t.Errorf("%s: got %v, want %v", tc.name, got, tc.want)
		}
	}
}

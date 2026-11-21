package query

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"AtlHyper/atlhyper_agent_v2/sdk"
	"AtlHyper/model_v3/log"

	_ "modernc.org/sqlite"
)

// ──────────────────────────────────────────────────────────────
// 日志概要：按服务的错误数
// ──────────────────────────────────────────────────────────────
//
// 主控用按服务的错误数给服务实体算 log_error_count，用 SeverityCounts["ERROR"] 给集群级实体算同名指标，
// 两边必须同一口径（同一时间窗、同一套「什么算 ERROR」）。ClickHouse 方言在 SQLite 上跑不了：
// 查询文本在这里守住，读取与组装用 SQLite 造出的结果行来测。

// 两条查询都由同一份时间窗与 ERROR 判定拼出
func TestLogSummaryQueries_SameWindowAndErrorRule(t *testing.T) {
	tests := []struct {
		name  string
		query string
		want  []string
	}{
		{"总计数（SeverityCounts）", logSummaryAggQuery, []string{"WHERE " + logSummaryWindow, "countIf(" + logErrorCondition + ") AS errors"}},
		{"按服务的错误数", serviceErrorsQuery, []string{"WHERE " + logSummaryWindow + " AND " + logErrorCondition}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, want := range tt.want {
				if !strings.Contains(tt.query, want) {
					t.Errorf("查询缺少 %q\nSQL:\n%s", want, tt.query)
				}
			}
		})
	}
}

// 按「命名空间 + 服务名」分组（命名空间与 APM 服务同一取法，主控才能对上同一个服务实体），
// 只取错误数最多的前 50 个；没有服务名的日志归不到任何服务，不计入
func TestServiceErrorsQuery_Shape(t *testing.T) {
	q := serviceErrorsQuery
	for _, want := range []string{
		"SELECT ServiceName, " + serviceNamespaceExpr("") + " AS ns, count() AS errors", // 列序即读取顺序
		"FROM otel_logs",
		"ServiceName != ''",
		"GROUP BY ServiceName, ns",
		"ORDER BY errors DESC",
		"LIMIT 50",
	} {
		if !strings.Contains(q, want) {
			t.Errorf("按服务的错误数查询缺少 %q\nSQL:\n%s", want, q)
		}
	}
	if n := strings.Count(q, "SeverityText"); n != 1 {
		t.Errorf("ERROR 判定只能来自共用的那一份，查询里出现 %d 处 SeverityText\nSQL:\n%s", n, q)
	}
}

// 读出按服务的错误数：按查询给出的顺序；查询成功但没有行时是空列表（⛔ nil）；解析不了的行跳过并记数
func TestScanServiceErrors(t *testing.T) {
	tests := []struct {
		name      string
		rows      string // 在 SQLite 上造结果行，列依次是服务名、命名空间、错误数
		want      []log.ServiceErrorCount
		wantSkips int
	}{
		{
			name: "按查询给出的顺序读出",
			rows: `SELECT 'checkout', 'shop', 42 UNION ALL SELECT 'worker', 'jobs', 3`,
			want: []log.ServiceErrorCount{
				{Namespace: "shop", Service: "checkout", Count: 42},
				{Namespace: "jobs", Service: "worker", Count: 3},
			},
		},
		{
			name: "没有出错的服务",
			rows: `SELECT 'checkout', 'shop', 42 WHERE 0`,
			want: []log.ServiceErrorCount{},
		},
		{
			name:      "解析不了的行跳过并记数",
			rows:      `SELECT 'checkout', 'shop', 'many' UNION ALL SELECT 'worker', 'jobs', 3`,
			want:      []log.ServiceErrorCount{{Namespace: "jobs", Service: "worker", Count: 3}},
			wantSkips: 1,
		},
	}
	db := sqliteDB(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rows, err := db.Query(tt.rows)
			if err != nil {
				t.Fatal(err)
			}
			defer rows.Close()

			skips := newScanSkips("test")
			got, err := scanServiceErrors(rows, skips)
			if err != nil {
				t.Fatalf("scanServiceErrors: %v", err)
			}
			if got == nil {
				t.Fatal("查询成功时应返回列表（可以为空），⛔ nil")
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got  %+v\nwant %+v", got, tt.want)
			}
			if skips.count != tt.wantSkips {
				t.Errorf("跳过 %d 行，want %d", skips.count, tt.wantSkips)
			}
		})
	}
}

// 日志概要带上按服务的错误数，与总的 ERROR 计数出自同一次 GetSummary；
// 按服务的查询失败时整份概要报错（与 Top 服务查询一致，调用方沿用上一份），⛔ 当成「没有服务出错」
func TestGetSummary_ServiceErrors(t *testing.T) {
	tests := []struct {
		name    string
		rows    string // 代替按服务查询的 SQLite 结果；空串 = 该查询失败
		want    []log.ServiceErrorCount
		wantErr bool
	}{
		{
			name: "带上按服务的错误数",
			rows: `SELECT 'checkout', 'shop', 42 UNION ALL SELECT 'worker', 'jobs', 3`,
			want: []log.ServiceErrorCount{
				{Namespace: "shop", Service: "checkout", Count: 42},
				{Namespace: "jobs", Service: "worker", Count: 3},
			},
		},
		{
			name: "没有出错的服务：空列表",
			rows: `SELECT 'checkout', 'shop', 42 WHERE 0`,
			want: []log.ServiceErrorCount{},
		},
		{
			name:    "按服务的查询失败",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := sqliteDB(t)
			mustExec(t, db, `CREATE TABLE agg(total INTEGER, errors INTEGER, warns INTEGER, infos INTEGER, debugs INTEGER, latest_at DATETIME)`)
			mustExec(t, db, `INSERT INTO agg VALUES (500, 45, 30, 400, 25, '2026-11-21 09:00:00')`)
			canned := map[string]string{
				logSummaryAggQuery:  `SELECT total, errors, warns, infos, debugs, latest_at FROM agg`,
				logTopServicesQuery: `SELECT 'checkout', 300`,
			}
			if tt.rows != "" {
				canned[serviceErrorsQuery] = tt.rows
			}
			client := &cannedClient{db: db, canned: canned}

			summary, err := NewLogQueryRepository(client).GetSummary(context.Background())
			if tt.wantErr {
				if err == nil {
					t.Fatalf("按服务的查询失败应返回错误，得到 %+v", summary)
				}
				return
			}
			if err != nil {
				t.Fatalf("GetSummary: %v", err)
			}
			if summary.SeverityCounts["ERROR"] != 45 {
				t.Errorf(`SeverityCounts["ERROR"] = %d, want 45`, summary.SeverityCounts["ERROR"])
			}
			if !reflect.DeepEqual(summary.ServiceErrors, tt.want) {
				t.Errorf("ServiceErrors = %#v\nwant %#v", summary.ServiceErrors, tt.want)
			}
			if args, asked := client.args[serviceErrorsQuery]; !asked || len(args) != 0 {
				t.Errorf("按服务的查询应执行且不带参数：asked=%v args=%v", asked, args)
			}
		})
	}
}

// cannedClient 按查询文本给出事先在 SQLite 上备好的结果，并记下每条查询收到的参数。
// ClickHouse 方言在 SQLite 上跑不了，这里验证的是读取与组装；没备结果的查询按失败处理
type cannedClient struct {
	sdk.ClickHouseClient
	db     *sql.DB
	canned map[string]string // ClickHouse 查询 → 代替它在 SQLite 上执行的查询
	args   map[string][]any
}

func (c *cannedClient) standIn(query string, args []any) (string, error) {
	if c.args == nil {
		c.args = map[string][]any{}
	}
	c.args[query] = args
	standIn, ok := c.canned[query]
	if !ok {
		return "", fmt.Errorf("没有备好结果的查询:\n%s", query)
	}
	return standIn, nil
}

func (c *cannedClient) Query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	standIn, err := c.standIn(query, args)
	if err != nil {
		return nil, err
	}
	return c.db.QueryContext(ctx, standIn)
}

func (c *cannedClient) QueryRow(ctx context.Context, query string, args ...any) *sql.Row {
	standIn, err := c.standIn(query, args)
	if err != nil {
		standIn = `SELECT 1 WHERE 0` // Scan 时得到 sql.ErrNoRows
	}
	return c.db.QueryRowContext(ctx, standIn)
}

// sqliteDB 用来造 *sql.Rows：ClickHouseClient 返回的就是它，没法直接伪造
func sqliteDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "rows.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func mustExec(t *testing.T, db *sql.DB, stmt string) {
	t.Helper()
	if _, err := db.Exec(stmt); err != nil {
		t.Fatalf("%s: %v", stmt, err)
	}
}

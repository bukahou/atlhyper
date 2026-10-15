// freshness.go 各信号最近一条数据的时间
//
// 页面上「没有流量」和「采集挂了」都表现为空白。三张表各取一次 max(时间列)
// 就能区分：metrics 是 Collector 主动拉取的，只要节点活着就有数据；
// traces / logs 由请求触发。metrics 还在流动而 traces 停了 = 没人访问；
// metrics 也停了 = 采集链路出问题。
package query

import (
	"context"
	"time"

	"AtlHyper/atlhyper_agent_v2/repository"
	"AtlHyper/atlhyper_agent_v2/sdk"
	"AtlHyper/common/logger"
	"AtlHyper/model_v3/cluster"
)

type freshnessRepository struct {
	client sdk.ClickHouseClient
}

// NewFreshnessQueryRepository 创建信号新鲜度查询仓库
func NewFreshnessQueryRepository(client sdk.ClickHouseClient) repository.FreshnessQueryRepository {
	return &freshnessRepository{client: client}
}

// 三张表的顺序与 SignalFreshness 的字段顺序一致
var freshnessSources = []string{"otel_metrics_gauge", "otel_traces", "otel_logs"}

// freshnessQuery 读 system.parts 的 max_time 而不是扫业务表。
//
// 旧写法 `SELECT max(TimeUnix) FROM otel_metrics_gauge WHERE TimeUnix > now()-24h`：
// 分区裁剪只到「天」，分区内主键（ServiceName, MetricName, Attributes, TimeUnix）
// 对纯时间条件不裁 ⇒ 实测每次扫 ≈910 万行 / 70 MiB，而它每 10s 跑一次、两个 agent 各跑。
// MergeTree 为每个 part 维护 min_time/max_time 元数据（分区键含 toDate 时填充），
// 三张表都是 PARTITION BY toDate(...)，读二十来行 system.parts 就是同一个答案。
// ⚠️ 精度到秒（新鲜度判定用不着纳秒）；若将来换成非 MergeTree 引擎需回退旧写法。
func freshnessQuery(table string) string {
	return "SELECT max(max_time) FROM system.parts" +
		" WHERE database = currentDatabase() AND table = '" + table + "' AND active"
}

// normalizeFreshnessTime 把「无 part」的 1970 归为零值。
//
// 空表上 max(max_time) 返回 1970-01-01 而不是 Go 零值；Master 只把零值判为 absent，
// 1970 会被当成「很久以前有数据」，与「从没有过数据」是两种不同的告警。
func normalizeFreshnessTime(ts time.Time) time.Time {
	if ts.Year() < 2000 {
		return time.Time{}
	}
	return ts
}

// GetSignalFreshness 取三个信号最近一条数据的时间。
//
// 单个信号查询失败时保持零值 —— Master 会把零值判为 absent，
// 这比编一个假时间戳好：查不到本身就是要显示给用户看的信息。
func (r *freshnessRepository) GetSignalFreshness(ctx context.Context) (*cluster.SignalFreshness, error) {
	out := &cluster.SignalFreshness{}
	targets := []*time.Time{&out.MetricsAt, &out.TracesAt, &out.LogsAt}

	for i, table := range freshnessSources {
		var ts time.Time
		if err := r.client.QueryRow(ctx, freshnessQuery(table)).Scan(&ts); err != nil {
			// 静默失败会让页面显示「无数据」却查不出原因 —— 这正是新鲜度要解决的问题本身
			logger.Warn("信号新鲜度查询失败", "table", table, "err", err)
			continue
		}
		*targets[i] = normalizeFreshnessTime(ts)
	}
	return out, nil
}

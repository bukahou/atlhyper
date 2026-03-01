# 跨信号关联 — Traces ↔ Logs 双向导航

> 状态：设计中
> 创建：2026-02-26
> 前置：APM 链路追踪（已完成）、日志查询（已完成）

## 1. 概要

AtlHyper 已实现 Traces、Logs、Metrics 三大可观测性支柱和 SLO，但各信号间相互孤立。本设计实现 **Traces ↔ Logs 双向关联导航**：

- **Trace → Logs**：在 Trace 瀑布图的 SpanDrawer 中查看该 Span 关联的日志
- **Logs → Trace**：在日志详情中点击 Trace ID 跳转到 APM 链路详情

桥接键：OTel 自动注入的 `TraceId` / `SpanId` 字段（日志和 Trace 共享）。

## 2. 背景

### 当前状态

| 信号 | 状态 | 关联能力 |
|------|------|----------|
| Traces | ClickHouse 存储，APM 页面展示 | 无 |
| Logs | ClickHouse 存储，日志页面展示 | 日志条目显示 TraceId/SpanId，但无跳转 |
| Metrics | 快照 + ClickHouse | 无 |

### 目标状态

```
APM Trace 详情                        日志页面
┌──────────────────┐                ┌──────────────────┐
│  SpanDrawer      │                │  日志列表         │
│  ┌────────────┐  │  ←──────────── │  [TraceId] 点击   │
│  │ 关联日志    │  │  Logs→Trace   │                  │
│  │ [查看日志]──┼──┼──────────────→│  ?traceId=xxx    │
│  └────────────┘  │  Trace→Logs   │  自动过滤         │
└──────────────────┘                └──────────────────┘
```

## 3. 数据流

### 3.1 Trace → Logs（SpanDrawer 内嵌日志）

```
SpanDrawer 选中 Span
  → 取 traceId + spanId + serviceName + timeWindow
  → 调用 queryLogs({ trace_id, span_id, service, since })
  → 前端 datasource/logs.ts 透传参数
  → API POST /api/v2/observe/logs/query { trace_id, span_id, ... }
  → Master observe.go LogsQuery()
    → 有 trace_id → 走 Command 路径（需 ClickHouse 精确查询）
  → Agent ch_query.go handleQueryLogs()
    → LogQueryOptions{ TraceId: "xxx", SpanId: "yyy" }
  → ClickHouse: WHERE TraceId = ? AND SpanId = ?
  → 返回关联日志列表
```

### 3.2 Logs → Trace（日志跳转 APM）

```
日志详情展示 TraceId
  → 用户点击「查看链路」按钮
  → router.push(`/observe/apm?trace=${traceId}`)
  → APM page.tsx 读取 searchParams.get("trace")
  → 自动进入 trace-detail 视图
  → 调用 getTraceDetail(traceId, clusterId)
```

## 4. 后端改动

### 4.1 Agent — LogQueryOptions 增加关联字段

**文件**: `atlhyper_agent_v2/repository/interfaces.go`（行 179-188）

```go
type LogQueryOptions struct {
    Query   string        // Body 全文搜索
    Service string        // ServiceName 过滤
    Level   string        // SeverityText 过滤
    Scope   string        // ScopeName 过滤
    TraceId string        // TraceId 精确匹配（新增）
    SpanId  string        // SpanId 精确匹配（新增）
    Limit   int           // 每页条数
    Offset  int           // 分页偏移
    Since   time.Duration // 时间范围
}
```

### 4.2 Agent — buildWhere() 增加 SQL 条件

**文件**: `atlhyper_agent_v2/repository/ch/query/log.go`（行 97-120）

在现有 `buildWhere()` 中追加：

```go
if opts.TraceId != "" {
    conditions = append(conditions, "TraceId = ?")
    args = append(args, opts.TraceId)
}
if opts.SpanId != "" {
    conditions = append(conditions, "SpanId = ?")
    args = append(args, opts.SpanId)
}
```

> ClickHouse otel_logs 表已有 `TraceId` 和 `SpanId` 列（OTel Collector 标准 schema），无需建表。

### 4.3 Agent — handleQueryLogs() 提取参数

**文件**: `atlhyper_agent_v2/service/command/ch_query.go`（行 97-114）

```go
opts := repository.LogQueryOptions{
    Query:   getStringParam(cmd.Params, "query"),
    Service: getStringParam(cmd.Params, "service"),
    Level:   getStringParam(cmd.Params, "level"),
    Scope:   getStringParam(cmd.Params, "scope"),
    TraceId: getStringParam(cmd.Params, "trace_id"),  // 新增
    SpanId:  getStringParam(cmd.Params, "span_id"),    // 新增
    Limit:   getIntParam(cmd.Params, "limit", 50),
    Offset:  getIntParam(cmd.Params, "offset", 0),
    Since:   getDurationParam(cmd.Params, "since", 15*time.Minute),
}
```

### 4.4 Master — LogsQuery 路由决策

**文件**: `atlhyper_master_v2/gateway/handler/observe.go`（LogsQuery 方法，行 330+）

当前逻辑：
- `query == ""` → 快照直读（RecentLogs）
- `query != ""` → Command 路径（ClickHouse）

修改后逻辑：
```
有 trace_id 或 span_id → 强制走 Command 路径（ClickHouse 精确查询）
有 query（全文搜索）   → Command 路径
其他                   → 快照直读
```

代码改动点（伪代码）：

```go
traceId := getBodyString(body, "trace_id")
spanId := getBodyString(body, "span_id")

// 快照路径：无全文搜索 且 无 trace/span 关联查询
if query == "" && traceId == "" && spanId == "" {
    // 现有快照直读逻辑不变...
}

// Command 路径：全文搜索 或 trace/span 关联查询
delete(body, "cluster_id")
h.executeQuery(w, r, clusterID, command.ActionQueryLogs, body, 0)
```

## 5. 前端改动

### 5.1 API 层 — queryLogs 增加参数

**文件**: `atlhyper_web/src/api/observe.ts`（行 146-160）

```typescript
export function queryLogs(params: {
  cluster_id: string;
  query?: string;
  service?: string;
  level?: string;
  scope?: string;
  trace_id?: string;   // 新增
  span_id?: string;    // 新增
  limit?: number;
  offset?: number;
  since?: string;
  start_time?: string;
  end_time?: string;
}) {
  return post<ObserveResponse<LogQueryResponse>>("/api/v2/observe/logs/query", params);
}
```

### 5.2 数据源层 — 透传关联参数

**文件**: `atlhyper_web/src/datasource/logs.ts`（行 1-72）

LogQueryParams 接口增加：

```typescript
export interface LogQueryParams extends MockLogQueryParams {
  clusterId?: string;
  startTime?: number;
  endTime?: number;
  traceId?: string;   // 新增
  spanId?: string;     // 新增
}
```

queryLogs 函数增加映射：

```typescript
const response = await observeApi.queryLogs({
  // ...existing params...
  trace_id: params.traceId,   // 新增
  span_id: params.spanId,     // 新增
});
```

### 5.3 APM 页面 — URL 参数驱动 Trace 详情

**文件**: `atlhyper_web/src/app/observe/apm/page.tsx`

读取 URL 参数，自动进入 trace-detail 视图：

```typescript
import { useSearchParams } from "next/navigation";

const searchParams = useSearchParams();
const traceParam = searchParams.get("trace");

// 初始化 ViewState 时：
useEffect(() => {
  if (traceParam && currentClusterId) {
    // 直接进入 trace-detail 视图
    getTraceDetail(traceParam, currentClusterId).then((detail) => {
      if (detail) {
        setView({
          level: "trace-detail",
          serviceName: detail.rootServiceName,
          operationName: detail.rootSpanName,
          traceId: traceParam,
          traceIndex: 0,
        });
        setTraceDetail(detail);
      }
    });
  }
}, [traceParam, currentClusterId]);
```

### 5.4 SpanDrawer — 启用 Logs 标签页

**文件**: `atlhyper_web/src/app/observe/apm/components/TraceWaterfall.tsx`

当前 logs 标签页被 `disabled={i > 0}` 禁用。改动：

1. 移除 logs 标签页的 disabled 限制
2. 在 logs 标签页内容中调用 `queryLogs({ traceId, spanId })` 加载关联日志
3. 渲染日志列表（复用日志页面的 LogEntry 组件）

```typescript
// Tabs 改动
{[t.timeline, t.metadata, t.logs].map((label, i) => (
  <button
    key={label}
    disabled={i === 1}  // 只禁用 metadata（尚未实现）
    onClick={() => setActiveTab(i)}
    // ...
  >
    {label}
  </button>
))}

// Logs 标签内容
{activeTab === 2 && selectedSpan && (
  <SpanLogs
    traceId={trace.traceId}
    spanId={selectedSpan.spanId}
    serviceName={selectedSpan.serviceName}
    startTime={selectedSpan.timestamp}
  />
)}
```

`SpanLogs` 组件：

```typescript
function SpanLogs({ traceId, spanId, serviceName, startTime }) {
  const [logs, setLogs] = useState<LogEntry[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    queryLogs({ traceId, spanId, clusterId: currentClusterId })
      .then(setLogs)
      .finally(() => setLoading(false));
  }, [traceId, spanId]);

  if (loading) return <Spinner />;
  if (logs.length === 0) return <EmptyState text={t.noCorrelatedLogs} />;
  return <LogList logs={logs} compact />;
}
```

### 5.5 日志页面 — 支持 URL 参数过滤 + 查看链路按钮

**文件**: `atlhyper_web/src/app/observe/logs/page.tsx`

1. 读取 URL 参数自动过滤：

```typescript
const searchParams = useSearchParams();
const traceIdParam = searchParams.get("traceId");
const spanIdParam = searchParams.get("spanId");

// 传入 queryLogs
const data = await queryLogs({
  clusterId: currentClusterId,
  traceId: traceIdParam || undefined,
  spanId: spanIdParam || undefined,
  // ...existing params...
});
```

2. 日志条目的 TraceId 字段增加「查看链路」按钮（如果 TraceId 非空）：

```typescript
{log.traceId && (
  <Link
    href={`/observe/apm?trace=${log.traceId}`}
    className="text-xs text-primary hover:underline"
  >
    {t.logs.viewTrace}
  </Link>
)}
```

### 5.6 i18n 增量

**现有键**（已定义但未使用）：
- `logs.traceId`: "Trace ID" / "Trace ID"
- `logs.spanId`: "Span ID" / "Span ID"
- `logs.viewTrace`: "查看链路" / "トレースを表示"

**新增键**：

| 键 | 中文 | 日文 |
|----|------|------|
| `apm.correlatedLogs` | "关联日志" | "関連ログ" |
| `apm.noCorrelatedLogs` | "该 Span 无关联日志" | "このSpanに関連ログがありません" |
| `apm.viewCorrelatedLogs` | "查看关联日志" | "関連ログを表示" |
| `logs.filteredByTrace` | "已按 Trace ID 过滤" | "Trace IDでフィルタリング中" |
| `logs.clearTraceFilter` | "清除过滤" | "フィルタをクリア" |

**文件变更**：
- `atlhyper_web/src/types/i18n.ts` — ApmTranslations 增加 3 个键
- `atlhyper_web/src/i18n/locales/zh.ts` — 对应中文
- `atlhyper_web/src/i18n/locales/ja.ts` — 对应日文

## 6. 文件变更清单

### 后端（4 个文件）

| 文件 | 操作 | 改动 |
|------|------|------|
| `atlhyper_agent_v2/repository/interfaces.go` | 修改 | LogQueryOptions 增加 TraceId/SpanId 字段 |
| `atlhyper_agent_v2/repository/ch/query/log.go` | 修改 | buildWhere() 增加 TraceId/SpanId SQL 条件 |
| `atlhyper_agent_v2/service/command/ch_query.go` | 修改 | handleQueryLogs() 提取 trace_id/span_id 参数 |
| `atlhyper_master_v2/gateway/handler/observe.go` | 修改 | LogsQuery 有 traceId 时强制走 Command 路径 |

### 前端（~8 个文件）

| 文件 | 操作 | 改动 |
|------|------|------|
| `atlhyper_web/src/api/observe.ts` | 修改 | queryLogs 增加 trace_id/span_id 参数 |
| `atlhyper_web/src/datasource/logs.ts` | 修改 | LogQueryParams 增加 traceId/spanId，透传 API |
| `atlhyper_web/src/app/observe/apm/page.tsx` | 修改 | useSearchParams 读取 ?trace= 参数 |
| `atlhyper_web/src/app/observe/apm/components/TraceWaterfall.tsx` | 修改 | 启用 logs 标签页，新增 SpanLogs 组件 |
| `atlhyper_web/src/app/observe/logs/page.tsx` | 修改 | 读取 ?traceId= 参数，日志条目增加跳转按钮 |
| `atlhyper_web/src/types/i18n.ts` | 修改 | ApmTranslations 增加关联日志键 |
| `atlhyper_web/src/i18n/locales/zh.ts` | 修改 | 新增中文翻译 |
| `atlhyper_web/src/i18n/locales/ja.ts` | 修改 | 新增日文翻译 |

### 合计：12 个文件修改，0 个新建

## 7. 实施阶段

### Phase 1：后端关联查询（4 个文件）

1. LogQueryOptions 增加 TraceId/SpanId
2. buildWhere() 增加 SQL 条件
3. handleQueryLogs() 提取参数
4. LogsQuery 路由决策修改
5. **验证**：`go build ./...` 通过

### Phase 2：前端 Trace → Logs（4 个文件）

1. API + 数据源层增加参数
2. SpanDrawer 启用 logs 标签页
3. 实现 SpanLogs 组件
4. i18n 新增键
5. **验证**：APM 页面 SpanDrawer 展示关联日志

### Phase 3：前端 Logs → Trace（2 个文件）

1. APM 页面读取 URL trace 参数
2. 日志页面增加「查看链路」跳转
3. **验证**：日志页面点击 TraceId 跳转到 APM 详情

## 8. 验证方法

### 后端验证

```bash
# 构建通过
go build ./atlhyper_agent_v2/...
go build ./atlhyper_master_v2/...

# 单元测试（如有）
go test ./atlhyper_agent_v2/repository/ch/query/ -run TestBuildWhere
```

### 前端验证

```bash
cd atlhyper_web && npm run build
```

### 端到端验证

1. **Trace → Logs**：
   - 打开 APM 页面 → 选择一条 Trace → 点击 Span → 切换到「日志」标签
   - 预期：展示该 Span 时间窗口内的关联日志（按 TraceId + SpanId 过滤）

2. **Logs → Trace**：
   - 打开日志页面 → 展开一条含 TraceId 的日志 → 点击「查看链路」
   - 预期：跳转到 APM 页面，自动加载对应 Trace 详情

3. **URL 直达**：
   - 访问 `/observe/apm?trace=abc123`
   - 预期：直接展示该 Trace 的瀑布图

4. **日志 URL 过滤**：
   - 访问 `/observe/logs?traceId=abc123`
   - 预期：日志列表已按 TraceId 过滤，显示过滤提示

# Topology 三视图改造设计

## 背景

当前拓扑页面全量展示所有节点（service / pod / node / ingress）和所有边，节点数量多时视觉噪音严重，难以区分内容。需要按关注点分层展示。

## 方案概述

在标题栏下方增加视图切换控件（SegmentedControl），提供三个视角：

| 视图 | 默认 | 展示内容 | 适用场景 |
|------|------|---------|---------|
| **Service** | 是 | 仅 service + ingress 节点，仅 `calls` + `routes_to` 边 | 看应用架构依赖 |
| **Anomaly** | 否 | 异常实体（rFinal > 0）+ 一跳邻居，所有关联边 | 事件调查 |
| **Full** | 否 | 全部节点和边，namespace combo 分组 | 全局探索 |

## 文件变更清单

| 文件 | 变更 |
|------|------|
| `types/i18n.ts` | AIOpsTranslations 新增 3 个视图名 + 1 个空状态翻译键 |
| `i18n/locales/zh.ts` | 新增对应中文翻译 |
| `i18n/locales/ja.ts` | 新增对应日文翻译 |
| `app/monitoring/topology/page.tsx` | 新增 viewMode 状态 + 过滤逻辑 + SegmentedControl UI + 更新图例 |
| `app/monitoring/topology/components/TopologyGraph.tsx` | 节点/边视觉增强 + Full 视图 namespace combo |

## 详细设计

### 1. i18n 新增键

```typescript
// types/i18n.ts — AIOpsTranslations 拓扑图部分追加
viewService: string;    // "服务视图"
viewAnomaly: string;    // "异常视图"
viewFull: string;       // "全量视图"
noAnomalies: string;    // "集群状态健康，暂无异常实体"
```

```typescript
// zh.ts
viewService: "服务视图",
viewAnomaly: "异常视图",
viewFull: "全量视图",
noAnomalies: "集群状态健康，暂无异常实体",
```

```typescript
// ja.ts
viewService: "サービスビュー",
viewAnomaly: "異常ビュー",
viewFull: "全体ビュー",
noAnomalies: "クラスタは正常です。異常エンティティはありません",
```

### 2. page.tsx 变更

#### 2.1 新增状态

```typescript
type ViewMode = "service" | "anomaly" | "full";
const [viewMode, setViewMode] = useState<ViewMode>("service");
```

#### 2.2 过滤逻辑（useMemo）

```typescript
const filteredGraph = useMemo(() => {
  if (!graph) return null;

  if (viewMode === "service") {
    // 只保留 service + ingress 节点
    const keepTypes = new Set(["service", "ingress"]);
    const keepNodes: Record<string, GraphNode> = {};
    for (const [key, node] of Object.entries(graph.nodes)) {
      if (keepTypes.has(node.type)) keepNodes[key] = node;
    }
    const keepKeys = new Set(Object.keys(keepNodes));
    // 只保留 calls + routes_to 且两端都在 keepKeys 中的边
    const keepEdgeTypes = new Set(["calls", "routes_to"]);
    const edges = graph.edges.filter(
      e => keepEdgeTypes.has(e.type) && keepKeys.has(e.from) && keepKeys.has(e.to)
    );
    return { ...graph, nodes: keepNodes, edges };
  }

  if (viewMode === "anomaly") {
    // 找出所有异常实体
    const anomalyKeys = new Set<string>();
    for (const r of entityRisks) {
      if (r.rFinal > 0) anomalyKeys.add(r.entityKey);
    }
    // 找一跳邻居
    const neighborKeys = new Set(anomalyKeys);
    for (const e of graph.edges) {
      if (anomalyKeys.has(e.from)) neighborKeys.add(e.to);
      if (anomalyKeys.has(e.to)) neighborKeys.add(e.from);
    }
    // 过滤节点
    const keepNodes: Record<string, GraphNode> = {};
    for (const [key, node] of Object.entries(graph.nodes)) {
      if (neighborKeys.has(key)) keepNodes[key] = node;
    }
    const keepKeys = new Set(Object.keys(keepNodes));
    // 过滤边：两端都在保留集中
    const edges = graph.edges.filter(
      e => keepKeys.has(e.from) && keepKeys.has(e.to)
    );
    return { ...graph, nodes: keepNodes, edges };
  }

  // full: 原样返回
  return graph;
}, [graph, entityRisks, viewMode]);
```

#### 2.3 SegmentedControl UI

放在标题栏与图表之间，替换当前的静态图例区域：

```tsx
{/* 视图切换 + 图例 */}
<div className="flex items-center justify-between flex-wrap gap-3">
  {/* 左侧: SegmentedControl */}
  <div className="flex rounded-lg border border-[var(--border-color)] overflow-hidden text-xs">
    {(["service", "anomaly", "full"] as const).map((mode) => (
      <button
        key={mode}
        onClick={() => setViewMode(mode)}
        className={`px-3 py-1.5 transition-colors ${
          viewMode === mode
            ? "bg-blue-500 text-white"
            : "bg-[var(--background)] text-muted hover:text-default"
        }`}
      >
        {t.aiops[`view${mode[0].toUpperCase()}${mode.slice(1)}`]}
      </button>
    ))}
  </div>

  {/* 右侧: 图例（保持现有图例，按当前 viewMode 调整展示） */}
  <div className="flex flex-wrap gap-3 text-xs text-muted">
    {/* Service 和 Full 视图展示形状图例 */}
    {/* 所有视图展示颜色图例 */}
    ...（保持现有图例内容）
  </div>
</div>
```

#### 2.4 Anomaly 空状态

```tsx
{viewMode === "anomaly" && Object.keys(filteredGraph.nodes).length === 0 && (
  <div className="flex flex-col items-center justify-center h-96 text-center">
    <CheckCircle className="w-12 h-12 mb-4 text-emerald-500" />
    <p className="text-default font-medium">{t.aiops.noAnomalies}</p>
  </div>
)}
```

#### 2.5 传递 filteredGraph 给 TopologyGraph

```tsx
<TopologyGraph
  graph={filteredGraph}    // 之前是 graph
  entityRisks={riskMap}
  selectedNode={selectedNode}
  onNodeSelect={setSelectedNode}
/>
```

### 3. TopologyGraph.tsx 视觉增强

在过滤逻辑之外，同步改善节点/边的可辨识度：

#### 3.1 节点增强

```typescript
// 当前
fillOpacity: 0.15,
lineWidth: 1,
labelFill: "#999",

// 改为
fillOpacity: 0.3,
lineWidth: 2,
labelFill: color,  // 跟随风险色
```

风险徽章门槛从 `rFinal > 50` 降为 `rFinal > 0`。

节点大小根据连接度数动态计算（Full 视图中更有用）：

```typescript
const degree = graph.edges.filter(e => e.from === n.key || e.to === n.key).length;
const baseSize = n.type === "node" ? 40 : n.type === "service" ? 34 : 26;
const size = baseSize + Math.min(degree * 2, 14);  // 动态范围 +0~14px
```

#### 3.2 边类型视觉区分

```typescript
function edgeStyle(type: string, isAnomaly: boolean) {
  if (isAnomaly) {
    return { stroke: "#ef4444", lineWidth: 2, strokeOpacity: 0.8, lineDash: undefined };
  }
  switch (type) {
    case "calls":     return { stroke: "#666", lineWidth: 1,   strokeOpacity: 0.5, lineDash: undefined };
    case "routes_to": return { stroke: "#888", lineWidth: 0.8, strokeOpacity: 0.4, lineDash: [6, 3] };
    case "selects":   return { stroke: "#888", lineWidth: 0.8, strokeOpacity: 0.3, lineDash: [3, 3] };
    case "runs_on":   return { stroke: "#888", lineWidth: 0.6, strokeOpacity: 0.2, lineDash: [2, 4] };
    default:          return { stroke: "#666", lineWidth: 0.8, strokeOpacity: 0.3, lineDash: undefined };
  }
}
```

异常判断扩展到双端：

```typescript
// 当前：只看源端 + 只看 calls 类型
const isAnomaly = e.type === "calls" && (entityRisks[e.from]?.rFinal ?? 0) > 50;

// 改为：源或目标任一端异常
const isAnomaly =
  (entityRisks[e.from]?.rFinal ?? 0) > 50 ||
  (entityRisks[e.to]?.rFinal ?? 0) > 50;
```

#### 3.3 Full 视图 namespace combo（G6 Hull 分组）

在 Full 视图下，按 namespace 对节点进行视觉分组。利用 G6 的 `hull` 插件：

```typescript
// 仅 Full 视图时追加 hull 插件
const namespaces = [...new Set(Object.values(graph.nodes).map(n => n.namespace).filter(Boolean))];
const hullPlugin = namespaces.map((ns, i) => ({
  type: "hull",
  members: Object.values(graph.nodes).filter(n => n.namespace === ns).map(n => n.key),
  labelText: ns,
  style: {
    fill: hullColors[i % hullColors.length],
    fillOpacity: 0.04,
    stroke: hullColors[i % hullColors.length],
    strokeOpacity: 0.15,
  },
}));
```

> 注：需要确认 G6 v5 的 hull 插件 API。如果 API 不支持，降级为不分组，后续迭代。

### 4. 各视图下图例展示

| 图例项 | Service | Anomaly | Full |
|--------|---------|---------|------|
| 形状图例 (Service/Pod/Node/Ingress) | 只展示 Service + Ingress | 全部 | 全部 |
| 颜色图例 (Healthy/Warning/Critical) | 全部 | 全部 | 全部 |
| 边类型图例 | 不展示 | 不展示 | 展示（实线=calls, 虚线=routes, 点线=selects, 稀疏点=runs_on） |

## 验证方法

```bash
cd atlhyper_web && npx next build
```

手动测试：
1. `/monitoring/topology` → 默认 Service 视图，只看到 service/ingress 节点和 calls/routes_to 边
2. 切换 Anomaly → 集群健康时显示空状态；有异常时只显示异常实体及邻居
3. 切换 Full → 全量展示，namespace 分组可见
4. 三个视图间切换流畅，选中节点状态保持

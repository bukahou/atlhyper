"use client";

import { useState, useEffect, useCallback, useMemo, useRef } from "react";
import { Layout } from "@/components/layout/Layout";
import { useI18n } from "@/i18n/context";
import { useClusterStore } from "@/store/clusterStore";
import { RefreshCw, Loader2, WifiOff, AlertTriangle, CheckCircle2, Pause, Play } from "lucide-react";

import { TopologyGraph } from "./components/TopologyGraph";
import { NodeDetail } from "./components/NodeDetail";

import { getGraph, getEntityRisks } from "@/api/aiops";
import type { DependencyGraph, EntityRisk, GraphNode } from "@/api/aiops";

type ViewMode = "service" | "anomaly" | "full";

const VIEW_MODES: ViewMode[] = ["anomaly", "service", "full"];

const VIEW_LABEL_KEYS: Record<ViewMode, "viewService" | "viewAnomaly" | "viewFull"> = {
  service: "viewService",
  anomaly: "viewAnomaly",
  full: "viewFull",
};

export default function TopologyPage() {
  const { t } = useI18n();
  const { currentClusterId } = useClusterStore();

  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [isRefreshing, setIsRefreshing] = useState(false);
  const [lastUpdate, setLastUpdate] = useState<Date>(new Date());

  const [graph, setGraph] = useState<DependencyGraph | null>(null);
  const [entityRisks, setEntityRisks] = useState<EntityRisk[]>([]);
  const [selectedNode, setSelectedNode] = useState<string | null>(null);
  const [viewMode, setViewMode] = useState<ViewMode>("anomaly");
  const [nsFilter, setNsFilter] = useState<Set<string>>(new Set());
  const [autoRefresh, setAutoRefresh] = useState(true);

  // entityRisks 数组 → map
  const riskMap = useMemo(() => {
    const map: Record<string, EntityRisk> = {};
    for (const r of entityRisks) {
      map[r.entityKey] = r;
    }
    return map;
  }, [entityRisks]);

  // 从图中提取所有 namespace（Full 视图筛选用）
  const allNamespaces = useMemo(() => {
    if (!graph) return [];
    const nsSet = new Set<string>();
    for (const node of Object.values(graph.nodes)) {
      if (node.namespace && node.namespace !== "_cluster") {
        nsSet.add(node.namespace);
      }
    }
    return [...nsSet].sort();
  }, [graph]);

  const toggleNs = useCallback((ns: string) => {
    setNsFilter((prev) => {
      const next = new Set(prev);
      if (next.has(ns)) next.delete(ns);
      else next.add(ns);
      return next;
    });
  }, []);

  // Full 视图默认选中第一个 namespace，避免全量展示节点过多
  // 仅在切换到 Full 视图时触发一次
  const fullInitedRef = useRef(false);
  useEffect(() => {
    if (viewMode === "full") {
      if (!fullInitedRef.current && allNamespaces.length > 0) {
        fullInitedRef.current = true;
        setNsFilter(new Set([allNamespaces[0]]));
      }
    } else {
      fullInitedRef.current = false;
      setNsFilter(new Set());
    }
  }, [viewMode, allNamespaces]);

  // 按视图模式过滤图数据
  const filteredGraph = useMemo(() => {
    if (!graph) return null;

    if (viewMode === "service") {
      const keepTypes = new Set(["service", "ingress"]);
      const keepNodes: Record<string, GraphNode> = {};
      for (const [key, node] of Object.entries(graph.nodes)) {
        if (keepTypes.has(node.type)) keepNodes[key] = node;
      }
      const keepKeys = new Set(Object.keys(keepNodes));
      const keepEdgeTypes = new Set(["calls", "routes_to"]);
      const edges = graph.edges.filter(
        (e) => keepEdgeTypes.has(e.type) && keepKeys.has(e.from) && keepKeys.has(e.to)
      );
      return { ...graph, nodes: keepNodes, edges };
    }

    if (viewMode === "anomaly") {
      const anomalyKeys = new Set<string>();
      for (const r of entityRisks) {
        if (r.rFinal > 0) anomalyKeys.add(r.entityKey);
      }
      // 一跳邻居
      const neighborKeys = new Set(anomalyKeys);
      for (const e of graph.edges) {
        if (anomalyKeys.has(e.from)) neighborKeys.add(e.to);
        if (anomalyKeys.has(e.to)) neighborKeys.add(e.from);
      }
      const keepNodes: Record<string, GraphNode> = {};
      for (const [key, node] of Object.entries(graph.nodes)) {
        if (neighborKeys.has(key)) keepNodes[key] = node;
      }
      const keepKeys = new Set(Object.keys(keepNodes));
      const edges = graph.edges.filter(
        (e) => keepKeys.has(e.from) && keepKeys.has(e.to)
      );
      return { ...graph, nodes: keepNodes, edges };
    }

    // full — 可选 namespace 筛选
    if (nsFilter.size > 0) {
      // 按选中 namespace 过滤节点
      const keepNodes: Record<string, GraphNode> = {};
      for (const [key, node] of Object.entries(graph.nodes)) {
        if (nsFilter.has(node.namespace)) keepNodes[key] = node;
      }
      // 保留与选中节点相连的 _cluster 节点（物理 Node）
      const keepKeys = new Set(Object.keys(keepNodes));
      for (const e of graph.edges) {
        if (keepKeys.has(e.from) && !keepKeys.has(e.to)) {
          const n = graph.nodes[e.to];
          if (n?.namespace === "_cluster") { keepNodes[e.to] = n; keepKeys.add(e.to); }
        }
        if (keepKeys.has(e.to) && !keepKeys.has(e.from)) {
          const n = graph.nodes[e.from];
          if (n?.namespace === "_cluster") { keepNodes[e.from] = n; keepKeys.add(e.from); }
        }
      }
      const edges = graph.edges.filter((e) => keepKeys.has(e.from) && keepKeys.has(e.to));
      return { ...graph, nodes: keepNodes, edges };
    }

    return graph;
  }, [graph, entityRisks, viewMode, nsFilter]);

  // 切换视图时，如果选中节点不在新视图中，清除选中
  useEffect(() => {
    if (selectedNode && filteredGraph && !filteredGraph.nodes[selectedNode]) {
      setSelectedNode(null);
    }
  }, [filteredGraph, selectedNode]);

  const loadData = useCallback(
    async (showLoading = true) => {
      if (!currentClusterId) return;
      if (showLoading) setIsRefreshing(true);

      try {
        const [graphData, risks] = await Promise.all([
          getGraph(currentClusterId),
          getEntityRisks(currentClusterId, "r_final", 100),
        ]);
        setGraph(graphData);
        setEntityRisks(risks);
        setError(null);
        setLastUpdate(new Date());
      } catch (err) {
        console.error("Failed to load topology:", err);
        setError(t.aiops.loadFailed);
      } finally {
        setLoading(false);
        setIsRefreshing(false);
      }
    },
    [currentClusterId, t.aiops.loadFailed]
  );

  useEffect(() => {
    loadData();
  }, [loadData]);

  // 自动刷新：默认 10s 间隔，仅更新风险数据不重建布局
  useEffect(() => {
    if (!autoRefresh) return;
    const interval = setInterval(() => loadData(false), 10_000);
    return () => clearInterval(interval);
  }, [autoRefresh, loadData]);

  if (loading) {
    return (
      <Layout>
        <div className="flex items-center justify-center h-96">
          <Loader2 className="w-8 h-8 animate-spin text-blue-500" />
        </div>
      </Layout>
    );
  }

  if (!currentClusterId) {
    return (
      <Layout>
        <div className="flex flex-col items-center justify-center h-96 text-center">
          <WifiOff className="w-12 h-12 mb-4 text-muted" />
          <p className="text-default font-medium mb-2">{t.aiops.noCluster}</p>
          <p className="text-sm text-muted">{t.aiops.noClusterDesc}</p>
        </div>
      </Layout>
    );
  }

  if (error && !graph) {
    return (
      <Layout>
        <div className="flex flex-col items-center justify-center h-96 text-center">
          <AlertTriangle className="w-12 h-12 mb-4 text-yellow-500" />
          <p className="text-default font-medium mb-2">{error}</p>
          <button
            onClick={() => loadData(true)}
            className="mt-4 px-4 py-2 bg-blue-500 text-white rounded-lg hover:bg-blue-600 transition-colors"
          >
            {t.aiops.retry}
          </button>
        </div>
      </Layout>
    );
  }

  const showEmptyAnomaly =
    viewMode === "anomaly" &&
    filteredGraph &&
    Object.keys(filteredGraph.nodes).length === 0;

  // Service 视图只展示 service/ingress 形状图例
  const showAllShapes = viewMode !== "service";

  return (
    <Layout>
      <div className="space-y-4 sm:space-y-6">
        {/* 标题栏 */}
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-lg sm:text-xl font-bold text-default">{t.aiops.topology}</h1>
            <p className="text-xs sm:text-sm text-muted mt-1">{t.aiops.dependencyGraph}</p>
          </div>
          <div className="flex items-center gap-2 sm:gap-3 flex-shrink-0">
            <span className="text-[10px] sm:text-xs text-muted hidden sm:block">
              {t.aiops.lastUpdate}: {lastUpdate.toLocaleTimeString()}
            </span>
            <button
              onClick={() => setAutoRefresh((v) => !v)}
              className={`p-2 rounded-lg hover:bg-[var(--hover-bg)] transition-colors ${
                autoRefresh ? "text-emerald-500" : "text-muted hover:text-default"
              }`}
              title={autoRefresh ? "Pause auto-refresh" : "Resume auto-refresh"}
            >
              {autoRefresh ? <Pause className="w-4 h-4" /> : <Play className="w-4 h-4" />}
            </button>
            <button
              onClick={() => loadData(true)}
              disabled={isRefreshing}
              className="p-2 rounded-lg hover:bg-[var(--hover-bg)] text-muted hover:text-default transition-colors disabled:opacity-50"
            >
              <RefreshCw className={`w-4 h-4 ${isRefreshing ? "animate-spin" : ""}`} />
            </button>
          </div>
        </div>

        {/* 视图切换 + 图例 */}
        <div className="flex items-center justify-between flex-wrap gap-3">
          {/* SegmentedControl */}
          <div className="flex rounded-lg border border-[var(--border-color)] overflow-hidden text-xs">
            {VIEW_MODES.map((mode) => (
              <button
                key={mode}
                onClick={() => setViewMode(mode)}
                className={`px-3 py-1.5 transition-colors ${
                  viewMode === mode
                    ? "bg-blue-500 text-white"
                    : "bg-[var(--background)] text-muted hover:text-default"
                }`}
              >
                {t.aiops[VIEW_LABEL_KEYS[mode]]}
              </button>
            ))}
          </div>

          {/* 图例 */}
          <div className="flex flex-wrap gap-3 text-xs text-muted">
            {/* 形状图例 */}
            <span className="flex items-center gap-1.5">
              <span className="w-3.5 h-3.5 rounded-full border-2 border-current inline-block" /> Service
            </span>
            <span className="flex items-center gap-1.5">
              <svg className="w-3.5 h-3.5" viewBox="-10 -10 20 20">
                <polygon points="0,-7 7,0 0,7 -7,0" fill="none" stroke="currentColor" strokeWidth={1.5} />
              </svg>
              Ingress
            </span>
            {showAllShapes && (
              <>
                <span className="flex items-center gap-1.5">
                  <span className="w-3.5 h-3.5 rounded-sm border-2 border-current inline-block" /> Pod
                </span>
                <span className="flex items-center gap-1.5">
                  <svg className="w-3.5 h-3.5" viewBox="-10 -10 20 20">
                    <polygon points="0,-8 7,4 -7,4" fill="none" stroke="currentColor" strokeWidth={1.5} />
                  </svg>
                  Node
                </span>
              </>
            )}
            <span className="mx-1 text-[var(--border-color)]">|</span>
            <span className="flex items-center gap-1.5">
              <span className="w-2.5 h-2.5 rounded-full bg-[#22c55e] inline-block" /> Healthy
            </span>
            <span className="flex items-center gap-1.5">
              <span className="w-2.5 h-2.5 rounded-full bg-[#eab308] inline-block" /> Warning
            </span>
            <span className="flex items-center gap-1.5">
              <span className="w-2.5 h-2.5 rounded-full bg-[#ef4444] inline-block" /> Critical
            </span>
          </div>
        </div>

        {/* Full 视图: Namespace 筛选 */}
        {viewMode === "full" && allNamespaces.length > 1 && (
          <div className="flex items-center gap-2 flex-wrap">
            <span className="text-xs text-muted">{t.common.namespace}:</span>
            {allNamespaces.map((ns) => {
              const isActive = nsFilter.size > 0 && nsFilter.has(ns);
              const isDimmed = nsFilter.size > 0 && !nsFilter.has(ns);
              return (
                <button
                  key={ns}
                  onClick={() => toggleNs(ns)}
                  className={`px-2.5 py-1 rounded-full text-xs transition-colors border ${
                    isActive
                      ? "bg-blue-500/15 text-blue-500 border-blue-500/30"
                      : isDimmed
                        ? "bg-[var(--background)] text-muted/40 border-[var(--border-color)] hover:text-muted"
                        : "bg-[var(--background)] text-muted border-[var(--border-color)] hover:text-default"
                  }`}
                >
                  {ns}
                </button>
              );
            })}
            {nsFilter.size > 0 && (
              <button
                onClick={() => setNsFilter(new Set())}
                className="px-2 py-1 text-xs text-muted hover:text-default transition-colors"
              >
                {t.common.reset}
              </button>
            )}
          </div>
        )}

        {/* Anomaly 空状态 */}
        {showEmptyAnomaly && (
          <div className="flex flex-col items-center justify-center h-96 text-center">
            <CheckCircle2 className="w-12 h-12 mb-4 text-emerald-500" />
            <p className="text-default font-medium">{t.aiops.noAnomalies}</p>
          </div>
        )}

        {/* 主体: 图 + 详情 */}
        {filteredGraph && !showEmptyAnomaly && (
          <div className="grid grid-cols-1 lg:grid-cols-[1fr_320px] gap-4" style={{ height: "calc(100vh - 300px)" }}>
            <TopologyGraph
              graph={filteredGraph}
              entityRisks={riskMap}
              selectedNode={selectedNode}
              onNodeSelect={setSelectedNode}
            />

            <div className="bg-card rounded-xl border border-[var(--border-color)] p-4 overflow-y-auto">
              {selectedNode ? (
                <NodeDetail entityKey={selectedNode} clusterId={currentClusterId} />
              ) : (
                <div className="flex items-center justify-center h-full text-sm text-muted">
                  {t.aiops.selectNode}
                </div>
              )}
            </div>
          </div>
        )}
      </div>
    </Layout>
  );
}

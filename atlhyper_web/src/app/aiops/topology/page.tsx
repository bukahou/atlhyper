"use client";

import { useState, useEffect, useCallback, useMemo, useRef } from "react";
import { Layout } from "@/components/layout/Layout";
import { useI18n } from "@/i18n/context";
import { useClusterStore } from "@/store/clusterStore";
import { RefreshCw, Loader2, WifiOff, AlertTriangle, CheckCircle2, Pause, Play } from "lucide-react";

import { TopologyGraph } from "./components/TopologyGraph";
import { NodeDetail } from "./components/NodeDetail";
import { TopologyToolbar } from "./components/TopologyToolbar";
import { useFilteredGraph } from "./components/useFilteredGraph";

import { getGraph, getEntityRisks } from "@/api/aiops";
import type { DependencyGraph, EntityRisk } from "@/api/aiops";

type ViewMode = "service" | "anomaly" | "full";

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

  // entityRisks 数组 -> map
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
  const filteredGraph = useFilteredGraph(graph, entityRisks, viewMode, nsFilter);

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

  // 自动刷新：默认 10s 间隔
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
              title={autoRefresh ? t.aiops.pauseAutoRefresh : t.aiops.resumeAutoRefresh}
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

        {/* 视图切换 + 图例 + Namespace 筛选 */}
        <TopologyToolbar
          viewMode={viewMode}
          onViewModeChange={setViewMode}
          allNamespaces={allNamespaces}
          nsFilter={nsFilter}
          onToggleNs={toggleNs}
          onResetNsFilter={() => setNsFilter(new Set())}
        />

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

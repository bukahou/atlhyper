"use client";

import { useState, useCallback, useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { Layout } from "@/components/layout/Layout";
import { QueryView } from "@/components/common";
import { useCurrentCluster } from "@/hooks/useCurrentCluster";
import { useManualRefetch } from "@/hooks/useManualRefetch";
import { useScopedState } from "@/hooks/useScopedState";
import { useI18n } from "@/i18n/context";
import { dependencyGraphQuery, entityRisksQuery } from "@/queries/aiops";
import { aiopsKeys } from "@/queries/keys";
import { REFRESH } from "@/queries/refresh";
import { RefreshCw, CheckCircle2, Pause, Play } from "lucide-react";

import { TopologyGraph } from "./components/TopologyGraph";
import { NodeDetail } from "./components/NodeDetail";
import { TopologyToolbar } from "./components/TopologyToolbar";
import { useFilteredGraph } from "./components/useFilteredGraph";

import type { EntityRisk } from "@/api/aiops";

type ViewMode = "service" | "anomaly" | "full";

/** 拓扑页给节点着色用的风险实体数：取风险最高的前 100 个 */
const TOPOLOGY_RISK_ENTITY_LIMIT = 100;
const NO_ENTITY_RISKS: EntityRisk[] = [];
const NO_NAMESPACE_FILTER = new Set<string>();

export default function TopologyPage() {
  const { t } = useI18n();
  const { clusterId } = useCurrentCluster();

  const [selectedNode, setSelectedNode] = useState<string | null>(null);
  const [viewMode, setViewMode] = useState<ViewMode>("anomaly");
  const [autoRefresh, setAutoRefresh] = useState(true);

  // 自动刷新：默认 10 秒；暂停后不再轮询，手动刷新照常可用
  const pollInterval = autoRefresh ? REFRESH.fast : false;
  const graphQuery = useQuery(dependencyGraphQuery(clusterId, pollInterval));
  const risksQuery = useQuery(entityRisksQuery(clusterId, TOPOLOGY_RISK_ENTITY_LIMIT, pollInterval));
  const { refetch: refreshTopology, isRefetching } = useManualRefetch(aiopsKeys.all);
  const graph = graphQuery.data ?? null;
  const entityRisks = risksQuery.data ?? NO_ENTITY_RISKS;

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

  // namespace 筛选依附于视图：换视图就作废。用户没动过筛选（null）时用该视图的默认值 ——
  // Full 视图默认只看第一个 namespace，避免全量展示节点过多；其余视图不筛选
  const [chosenNamespaces, setChosenNamespaces] = useScopedState<Set<string> | null>(viewMode, null);
  const firstNamespace = allNamespaces[0];
  const defaultNamespaces = useMemo(
    () => (viewMode === "full" && firstNamespace !== undefined ? new Set([firstNamespace]) : NO_NAMESPACE_FILTER),
    [viewMode, firstNamespace],
  );
  const nsFilter = chosenNamespaces ?? defaultNamespaces;

  const toggleNs = useCallback(
    (ns: string) => {
      const next = new Set(nsFilter);
      if (next.has(ns)) next.delete(ns);
      else next.add(ns);
      setChosenNamespaces(next);
    },
    [nsFilter, setChosenNamespaces],
  );

  // 按视图模式过滤图数据
  const filteredGraph = useFilteredGraph(graph, entityRisks, viewMode, nsFilter);

  // 选中的节点不在当前视图里（切了视图、或它从图里消失了）就当作没有选中
  const visibleSelectedNode = selectedNode && filteredGraph?.nodes[selectedNode] ? selectedNode : null;

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
            {graphQuery.dataUpdatedAt > 0 && (
              <span className="text-[10px] sm:text-xs text-muted hidden sm:block">
                {t.aiops.lastUpdate}: {new Date(graphQuery.dataUpdatedAt).toLocaleTimeString()}
              </span>
            )}
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
              onClick={refreshTopology}
              disabled={isRefetching}
              title={t.common.refresh}
              className="p-2 rounded-lg hover:bg-[var(--hover-bg)] text-muted hover:text-default transition-colors disabled:opacity-50"
            >
              <RefreshCw className={`w-4 h-4 ${isRefetching ? "animate-spin" : ""}`} />
            </button>
          </div>
        </div>

        <QueryView query={graphQuery} requireCluster>
          {() => (
            <>
              {/* 视图切换 + 图例 + Namespace 筛选 */}
              <TopologyToolbar
                viewMode={viewMode}
                onViewModeChange={setViewMode}
                allNamespaces={allNamespaces}
                nsFilter={nsFilter}
                onToggleNs={toggleNs}
                onResetNsFilter={() => setChosenNamespaces(NO_NAMESPACE_FILTER)}
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
                <div
                  className="grid grid-cols-1 lg:grid-cols-[1fr_320px] gap-4"
                  style={{ height: "calc(100vh - 300px)" }}
                >
                  <TopologyGraph
                    graph={filteredGraph}
                    entityRisks={riskMap}
                    selectedNode={visibleSelectedNode}
                    onNodeSelect={setSelectedNode}
                  />

                  <div className="bg-card rounded-xl border border-[var(--border-color)] p-4 overflow-y-auto">
                    {visibleSelectedNode ? (
                      <NodeDetail entityKey={visibleSelectedNode} clusterId={clusterId} />
                    ) : (
                      <div className="flex items-center justify-center h-full text-sm text-muted">
                        {t.aiops.selectNode}
                      </div>
                    )}
                  </div>
                </div>
              )}
            </>
          )}
        </QueryView>
      </div>
    </Layout>
  );
}

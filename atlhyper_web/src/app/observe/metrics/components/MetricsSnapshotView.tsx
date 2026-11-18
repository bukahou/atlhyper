"use client";

import { useMemo } from "react";
import { AlertTriangle, Server } from "lucide-react";
import { useI18n } from "@/i18n/context";
import type { ClusterNodeMetricsSnapshot } from "@/datasource/metrics";
import type { HardwareHealth } from "@/types/hardware";
import { ClusterOverviewChart } from "./ClusterOverviewChart";
import { HardwareMatrix } from "./HardwareMatrix";
import { HardwareSummaryTiles } from "./HardwareSummaryTiles";
import { SummaryCard } from "./SummaryCard";

interface MetricsSnapshotViewProps {
  snapshot: ClusterNodeMetricsSnapshot;
  /** 硬件健康矩阵；接口还没返回或失败时为 null，各组件自己显示「无数据」 */
  hardware: HardwareHealth | null;
  clusterId: string;
  expandedNode: string | null;
  onNodeToggle: (nodeName: string) => void;
  /** 趋势图取最近多少小时的历史 */
  historyHours: number;
  /** 趋势图时间窗（毫秒） */
  trendSpanMs: number;
}

/** 指标页的主体：快照带 + 节点矩阵 + 集群趋势。只在快照取到之后渲染 */
export function MetricsSnapshotView({
  snapshot,
  hardware,
  clusterId,
  expandedNode,
  onNodeToggle,
  historyHours,
  trendSpanMs,
}: MetricsSnapshotViewProps) {
  const { t } = useI18n();
  const nm = t.nodeMetrics;
  const { summary, nodes } = snapshot;

  // 计算告警节点数
  const warningNodes = useMemo(() => {
    return nodes.filter((node) => {
      return (
        node.cpu.usagePct >= 80 ||
        node.memory.usagePct >= 80 ||
        node.temperature.cpuTempC >= 75
      );
    }).length;
  }, [nodes]);

  return (
    <div className="space-y-4 sm:space-y-6">
      {/* 快照带：集群规模/告警 + 硬件速览，合并为一排（只留有判定语义的）
          —— 平均 CPU/内存两张纯快照已删：矩阵每行有精确值，均值无行动价值 */}
      <div className="grid grid-cols-2 sm:grid-cols-4 lg:grid-cols-7 gap-3">
        {summary && (
          <>
            <SummaryCard
              icon={Server}
              label={nm.summary.nodes}
              value={`${summary.onlineNodes}/${summary.totalNodes}`}
              subValue={`${warningNodes} ${nm.summary.warnings}`}
              color="bg-indigo-500/10 text-indigo-500"
            />
            <SummaryCard
              icon={AlertTriangle}
              label={nm.summary.warnings}
              value={warningNodes.toString()}
              subValue={nm.summary.nodesNeedAttention}
              color={warningNodes > 0 ? "bg-yellow-500/10 text-yellow-500" : "bg-emerald-500/10 text-emerald-500"}
            />
          </>
        )}
        <HardwareSummaryTiles data={hardware} bare />
      </div>

      {/* 统一节点表：判定矩阵 + 行点击下钻（原 NodeCard 详情体） */}
      {nodes.length === 0 ? (
        <div className="text-center py-12 bg-card rounded-xl border border-[var(--border-color)]">
          <Server className="w-12 h-12 mx-auto mb-3 text-muted opacity-50" />
          <p className="text-default font-medium mb-2">{nm.noMetricsData}</p>
          <p className="text-sm text-muted">{nm.noMetricsDesc}</p>
        </div>
      ) : (
        <HardwareMatrix
          data={hardware}
          nodes={nodes}
          expandedNode={expandedNode}
          onToggle={onNodeToggle}
          historyHours={historyHours}
          spanMs={trendSpanMs}
        />
      )}

      {/* 集群趋势（时间范围跟随页面级选择器） */}
      {nodes.length > 1 && (
        <ClusterOverviewChart nodes={nodes} clusterId={clusterId} hours={historyHours} />
      )}
    </div>
  );
}

"use client";

import { useState, useCallback, useMemo } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import { Layout } from "@/components/layout/Layout";
import { useI18n } from "@/i18n/context";
import { PageHeader, QueryView, StatsCard, DataTable, ConfirmDialog } from "@/components/common";
import { useCurrentCluster } from "@/hooks/useCurrentCluster";
import { useDelayedInvalidate } from "@/hooks/useDelayedInvalidate";
import { useRequireAuth } from "@/hooks/useRequireAuth";
import { podOverviewQuery } from "@/queries/cluster";
import { restartPodMutation } from "@/queries/cluster-mutations";
import { clusterKeys } from "@/queries/keys";
import { REFRESH } from "@/queries/refresh";
import type { PodItem } from "@/types/cluster";
import { PodDetailModal, PodLogsViewer } from "@/components/pod";
import { PodFilterBar, type PodFilters } from "./components/PodFilterBar";
import { getPodColumns } from "./components/PodTableColumns";

export default function PodPage() {
  const { t } = useI18n();
  const requireAuth = useRequireAuth();
  const { clusterId } = useCurrentCluster();
  const query = useQuery(podOverviewQuery(clusterId));
  const data = query.data;
  const restart = useMutation(restartPodMutation(clusterId));
  const invalidateAfterCommand = useDelayedInvalidate();

  // 筛选状态
  const [filters, setFilters] = useState<PodFilters>({
    namespace: "",
    node: "",
    status: "",
    search: "",
  });

  // 详情弹窗状态
  const [selectedPod, setSelectedPod] = useState<PodItem | null>(null);
  const [detailOpen, setDetailOpen] = useState(false);

  // 日志查看器状态
  const [logsOpen, setLogsOpen] = useState(false);
  const [logsContainer, setLogsContainer] = useState("");

  // 重启确认状态
  const [restartTarget, setRestartTarget] = useState<PodItem | null>(null);

  // 打开详情
  const handleRowClick = useCallback((pod: PodItem) => {
    setSelectedPod(pod);
    setDetailOpen(true);
  }, []);

  // 从详情页打开日志（需要 pod.logs.read）
  const handleViewLogs = (containerName: string) => {
    requireAuth(() => {
      setLogsContainer(containerName);
      setLogsOpen(true);
    }, "pod.logs.read");
  };

  // 显示重启确认（需要 cluster.operate）
  const handleRestartClick = useCallback(
    (pod: PodItem) => {
      requireAuth(() => setRestartTarget(pod));
    },
    [requireAuth],
  );

  // 确认重启
  const handleRestartConfirm = () => {
    if (!restartTarget) return;
    restart.mutate(
      { namespace: restartTarget.namespace, name: restartTarget.name },
      {
        onSuccess: () => {
          setRestartTarget(null);
          // 重启指令由 Agent 异步执行：下发成功时列表已刷新过一次，
          // 晚一点再刷新一次才看得到 Pod 重建后的状态
          invalidateAfterCommand(clusterKeys.resourceScope("pod"));
        },
        onError: (err) => console.error("Restart failed:", err),
      },
    );
  };

  // 提取唯一的 namespaces 和 nodes
  const { namespaces, nodes } = useMemo(() => {
    const pods = data?.pods || [];
    const nsSet = new Set<string>();
    const nodeSet = new Set<string>();

    pods.forEach((pod) => {
      if (pod.namespace) nsSet.add(pod.namespace);
      if (pod.node) nodeSet.add(pod.node);
    });

    return {
      namespaces: Array.from(nsSet).sort(),
      nodes: Array.from(nodeSet).sort(),
    };
  }, [data?.pods]);

  // 根据筛选条件过滤数据
  const filteredPods = useMemo(() => {
    const pods = data?.pods || [];

    return pods.filter((pod) => {
      if (filters.search && !pod.name.toLowerCase().includes(filters.search.toLowerCase())) {
        return false;
      }
      if (filters.namespace && pod.namespace !== filters.namespace) {
        return false;
      }
      if (filters.node && pod.node !== filters.node) {
        return false;
      }
      if (filters.status && pod.phase !== filters.status) {
        return false;
      }
      return true;
    });
  }, [data?.pods, filters]);

  const handleFilterChange = (key: string, value: string) => {
    setFilters((prev) => ({ ...prev, [key]: value }));
  };

  const columns = useMemo(
    () => getPodColumns(t, handleRowClick, handleRestartClick),
    [t, handleRowClick, handleRestartClick],
  );

  const totalPods = data ? (data.cards.running ?? 0) + (data.cards.pending ?? 0) + (data.cards.failed ?? 0) + (data.cards.unknown ?? 0) : 0;

  return (
    <Layout>
      <div className="space-y-4">
        <PageHeader
          title={t.nav.pod}
          description={t.pod.pageDescription}
          autoRefreshSeconds={REFRESH.standard / 1000}
        />

        {data && (
          <div className="grid grid-cols-2 md:grid-cols-5 gap-4">
            <StatsCard label={t.common.total} value={totalPods} />
            <StatsCard label={t.status.running} value={data.cards.running ?? 0} iconColor="text-green-500" />
            <StatsCard label={t.status.pending} value={data.cards.pending ?? 0} iconColor="text-yellow-500" />
            <StatsCard label={t.status.failed} value={data.cards.failed ?? 0} iconColor="text-red-500" />
            <StatsCard label={t.status.unknown} value={data.cards.unknown ?? 0} iconColor="text-gray-500" />
          </div>
        )}

        {/* 筛选栏 */}
        <PodFilterBar
          namespaces={namespaces}
          nodes={nodes}
          filters={filters}
          onFilterChange={handleFilterChange}
        />

        {/* 数据表格 */}
        <QueryView query={query} requireCluster>
          {() => (
            <div className="bg-card rounded-xl border border-[var(--border-color)] overflow-hidden">
              <DataTable
                columns={columns}
                data={filteredPods}
                loading={false}
                error=""
                keyExtractor={(pod, index) => `${index}-${pod.namespace}/${pod.name}`}
                onRowClick={handleRowClick}
                pageSize={10}
              />
            </div>
          )}
        </QueryView>
      </div>

      {/* Pod 详情弹窗 */}
      {selectedPod && (
        <PodDetailModal
          isOpen={detailOpen}
          onClose={() => setDetailOpen(false)}
          namespace={selectedPod.namespace}
          podName={selectedPod.name}
          onViewLogs={handleViewLogs}
        />
      )}

      {/* 日志查看器 */}
      {selectedPod && (
        <PodLogsViewer
          isOpen={logsOpen}
          onClose={() => setLogsOpen(false)}
          namespace={selectedPod.namespace}
          podName={selectedPod.name}
          containerName={logsContainer}
        />
      )}

      {/* 重启确认对话框 */}
      <ConfirmDialog
        isOpen={!!restartTarget}
        onClose={() => setRestartTarget(null)}
        onConfirm={handleRestartConfirm}
        title={t.pod.restartConfirmTitle}
        message={t.pod.restartConfirmMessage.replace("{name}", restartTarget?.name || "")}
        confirmText={t.pod.restart}
        cancelText={t.common.cancel}
        loading={restart.isPending}
        variant="warning"
      />
    </Layout>
  );
}
